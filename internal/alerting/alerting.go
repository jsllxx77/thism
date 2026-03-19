package alerting

import (
	"fmt"
	"math"
	"time"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

type Sender interface {
	Send(settings models.NotificationSettings, event models.AlertEvent) error
}

type Evaluator struct {
	Store  *store.Store
	Sender Sender
}

type eventState struct {
	metric    models.ResourceMetric
	value     float64
	warning   float64
	critical  float64
	severity  models.AlertSeverity
	threshold float64
	triggered bool
}

func (e *Evaluator) Process(node *models.Node, metrics *models.MetricsPayload) error {
	if e == nil || e.Store == nil || e.Sender == nil || node == nil || metrics == nil {
		return nil
	}
	settings, err := e.Store.GetNotificationSettings()
	if err != nil {
		return err
	}
	if !settings.Enabled || settings.Channel != string(models.NotificationChannelTelegram) || len(settings.TelegramTargets) == 0 {
		return nil
	}

	states := evaluateStates(metrics, settings)
	alerts, recoveries := buildEvents(node, states, metrics)
	for _, event := range alerts {
		allowed, err := e.Store.ShouldSendAlert(event.NodeID, string(event.Metric), string(event.Severity), time.Duration(settings.CooldownMinutes)*time.Minute, event.ObservedAt)
		if err != nil {
			return err
		}
		if !allowed {
			continue
		}
		if err := e.Sender.Send(settings, event); err != nil {
			return err
		}
		if err := e.Store.RecordAlertDelivery(event.NodeID, string(event.Metric), string(event.Severity), event.Value, event.Threshold, event.ObservedAt); err != nil {
			return err
		}
	}
	for _, event := range recoveries {
		active, err := e.Store.HasActiveAlertDelivery(event.NodeID, string(event.Metric))
		if err != nil {
			return err
		}
		if !active {
			continue
		}
		if err := e.Sender.Send(settings, event); err != nil {
			return err
		}
		if err := e.Store.ClearAlertDelivery(event.NodeID, string(event.Metric)); err != nil {
			return err
		}
	}
	return nil
}

func evaluateStates(metrics *models.MetricsPayload, settings models.NotificationSettings) []eventState {
	values := []eventState{
		{metric: models.ResourceMetricCPU, value: clampPercent(metrics.CPU), warning: settings.CPUWarningPercent, critical: settings.CPUCriticalPercent},
		{metric: models.ResourceMetricMemory, value: memoryPercent(metrics), warning: settings.MemWarningPercent, critical: settings.MemCriticalPercent},
		{metric: models.ResourceMetricDisk, value: diskPercent(metrics), warning: settings.DiskWarningPercent, critical: settings.DiskCriticalPercent},
	}
	for i := range values {
		values[i].severity, values[i].threshold, values[i].triggered = classify(values[i].value, values[i].warning, values[i].critical)
	}
	return values
}

func buildEvents(node *models.Node, states []eventState, metrics *models.MetricsPayload) ([]models.AlertEvent, []models.AlertEvent) {
	observedAt := metrics.TS
	if observedAt <= 0 {
		observedAt = time.Now().Unix()
	}
	nodeName := firstNonEmpty(node.Name, node.ID)
	alerts := make([]models.AlertEvent, 0, len(states))
	recoveries := make([]models.AlertEvent, 0, len(states))
	for _, state := range states {
		if state.triggered {
			alerts = append(alerts, models.AlertEvent{NodeID: node.ID, NodeName: nodeName, Metric: state.metric, Severity: state.severity, Value: state.value, Threshold: state.threshold, ObservedAt: observedAt})
			continue
		}
		recoveries = append(recoveries, models.AlertEvent{NodeID: node.ID, NodeName: nodeName, Metric: state.metric, Severity: models.AlertSeverityResolved, Value: state.value, Threshold: 0, ObservedAt: observedAt})
	}
	return alerts, recoveries
}

func classify(value, warning, critical float64) (models.AlertSeverity, float64, bool) {
	if critical > 0 && value >= critical {
		return models.AlertSeverityCritical, critical, true
	}
	if warning > 0 && value >= warning {
		return models.AlertSeverityWarning, warning, true
	}
	return "", 0, false
}

func memoryPercent(metrics *models.MetricsPayload) float64 {
	if metrics.Mem.Total == 0 {
		return 0
	}
	return clampPercent((float64(metrics.Mem.Used) / float64(metrics.Mem.Total)) * 100)
}

func diskPercent(metrics *models.MetricsPayload) float64 {
	var used uint64
	var total uint64
	for _, disk := range metrics.Disk {
		used += disk.Used
		total += disk.Total
	}
	if total == 0 {
		return 0
	}
	return clampPercent((float64(used) / float64(total)) * 100)
}

func clampPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return math.Round(value*10) / 10
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return fmt.Sprintf("node")
}
