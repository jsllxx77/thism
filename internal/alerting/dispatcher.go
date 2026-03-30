package alerting

import (
	"log"
	"sync"
	"time"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

type dispatchJobKind int

const (
	dispatchJobMetrics dispatchJobKind = iota + 1
	dispatchJobHeartbeat
)

type dispatchJob struct {
	kind       dispatchJobKind
	node       models.Node
	metrics    models.MetricsPayload
	online     bool
	observedAt int64
}

const DefaultDispatcherQueueCapacity = 256
const DefaultDispatcherDropLogInterval = time.Minute

type DispatcherStats struct {
	Capacity      int
	QueueDepth    int
	HighWatermark int
	Enqueued      uint64
	Processed     uint64
	Dropped       uint64
}

type DispatcherRuntimeStats struct {
	ActiveDispatchers int    `json:"active_dispatchers"`
	TotalCapacity     int    `json:"total_capacity"`
	QueueDepth        int    `json:"queue_depth"`
	HighWatermark     int    `json:"high_watermark"`
	Enqueued          uint64 `json:"enqueued"`
	Processed         uint64 `json:"processed"`
	Dropped           uint64 `json:"dropped"`
}

var dispatcherRuntimeStats struct {
	mu    sync.Mutex
	stats DispatcherRuntimeStats
}

// Dispatcher moves alert evaluation and notification delivery off the agent
// websocket hot path while preserving in-order processing per connection.
type Dispatcher struct {
	mu               sync.Mutex
	cond             *sync.Cond
	queue            []dispatchJob
	queueCapacity    int
	closed           bool
	closeOnce        sync.Once
	stats            DispatcherStats
	lastDropLogAt    time.Time
	droppedAtLastLog uint64
	now              func() time.Time
	logf             func(string, ...any)
	dropLogInterval  time.Duration
	wg               sync.WaitGroup
	evaluator        *Evaluator
}

func NewDispatcher(st *store.Store, sender Sender) *Dispatcher {
	return NewDispatcherWithCapacity(st, sender, DefaultDispatcherQueueCapacity)
}

func NewDispatcherWithCapacity(st *store.Store, sender Sender, capacity int) *Dispatcher {
	if capacity <= 0 {
		capacity = DefaultDispatcherQueueCapacity
	}

	dispatcher := &Dispatcher{
		queueCapacity:   capacity,
		evaluator:       &Evaluator{Store: st, Sender: sender},
		now:             time.Now,
		logf:            log.Printf,
		dropLogInterval: DefaultDispatcherDropLogInterval,
		stats: DispatcherStats{
			Capacity: capacity,
		},
	}
	dispatcher.cond = sync.NewCond(&dispatcher.mu)
	recordDispatcherCreated(capacity)
	dispatcher.wg.Add(1)
	go dispatcher.run()
	return dispatcher
}

func (d *Dispatcher) Close() {
	if d == nil {
		return
	}

	d.closeOnce.Do(func() {
		d.mu.Lock()
		d.closed = true
		d.cond.Broadcast()
		d.mu.Unlock()
		d.wg.Wait()
		recordDispatcherClosed(d.queueCapacity)
	})
}

func (d *Dispatcher) EnqueueMetrics(node *models.Node, metrics *models.MetricsPayload) bool {
	if d == nil || node == nil || metrics == nil {
		return false
	}

	return d.enqueue(dispatchJob{
		kind:    dispatchJobMetrics,
		node:    cloneAlertNode(node),
		metrics: cloneAlertMetrics(metrics),
	})
}

func (d *Dispatcher) EnqueueHeartbeat(node *models.Node, online bool, observedAt int64) bool {
	if d == nil || node == nil {
		return false
	}

	return d.enqueue(dispatchJob{
		kind:       dispatchJobHeartbeat,
		node:       cloneAlertNode(node),
		online:     online,
		observedAt: observedAt,
	})
}

func (d *Dispatcher) enqueue(job dispatchJob) bool {
	d.mu.Lock()
	if d.closed {
		d.stats.Dropped += 1
		recordDispatcherDropped()
		d.mu.Unlock()
		return false
	}
	if len(d.queue) >= d.queueCapacity {
		d.stats.Dropped += 1
		recordDispatcherDropped()
		shouldLog, droppedSinceLastLog, totalDropped, queueDepth, capacity := d.shouldLogDropLocked()
		logf := d.logf
		d.mu.Unlock()
		if shouldLog && logf != nil {
			logf("alert dispatcher: dropping queued jobs due to full queue (dropped_since_last_log=%d total_dropped=%d queue_depth=%d capacity=%d)", droppedSinceLastLog, totalDropped, queueDepth, capacity)
		}
		return false
	}

	d.queue = append(d.queue, job)
	d.stats.Enqueued += 1
	if len(d.queue) > d.stats.HighWatermark {
		d.stats.HighWatermark = len(d.queue)
	}
	recordDispatcherEnqueue(len(d.queue))
	d.cond.Signal()
	d.mu.Unlock()
	return true
}

func (d *Dispatcher) shouldLogDropLocked() (bool, uint64, uint64, int, int) {
	now := time.Now()
	if d != nil && d.now != nil {
		now = d.now()
	}
	interval := d.dropLogInterval
	if interval <= 0 {
		interval = DefaultDispatcherDropLogInterval
	}
	if !d.lastDropLogAt.IsZero() && now.Sub(d.lastDropLogAt) < interval {
		return false, 0, 0, 0, 0
	}

	totalDropped := d.stats.Dropped
	droppedSinceLastLog := totalDropped - d.droppedAtLastLog
	if droppedSinceLastLog == 0 {
		droppedSinceLastLog = 1
	}
	d.lastDropLogAt = now
	d.droppedAtLastLog = totalDropped
	return true, droppedSinceLastLog, totalDropped, len(d.queue), d.queueCapacity
}

func (d *Dispatcher) run() {
	defer d.wg.Done()

	for {
		job, ok := d.dequeue()
		if !ok {
			return
		}

		switch job.kind {
		case dispatchJobMetrics:
			if err := d.evaluator.Process(&job.node, &job.metrics); err != nil {
				log.Printf("alert dispatcher: process metrics for node %s failed: %v", job.node.ID, err)
			}
		case dispatchJobHeartbeat:
			if err := d.evaluator.ProcessHeartbeat(&job.node, job.online, job.observedAt); err != nil {
				log.Printf("alert dispatcher: process heartbeat for node %s failed: %v", job.node.ID, err)
			}
		}

		d.mu.Lock()
		d.stats.Processed += 1
		currentQueueDepth := len(d.queue)
		d.mu.Unlock()
		recordDispatcherProcessed(currentQueueDepth)
	}
}

func (d *Dispatcher) dequeue() (dispatchJob, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for len(d.queue) == 0 && !d.closed {
		d.cond.Wait()
	}
	if len(d.queue) == 0 {
		return dispatchJob{}, false
	}

	job := d.queue[0]
	copy(d.queue, d.queue[1:])
	d.queue[len(d.queue)-1] = dispatchJob{}
	d.queue = d.queue[:len(d.queue)-1]
	return job, true
}

func (d *Dispatcher) Stats() DispatcherStats {
	if d == nil {
		return DispatcherStats{}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	stats := d.stats
	stats.QueueDepth = len(d.queue)
	return stats
}

func DispatcherRuntimeStatsSnapshot() DispatcherRuntimeStats {
	dispatcherRuntimeStats.mu.Lock()
	defer dispatcherRuntimeStats.mu.Unlock()
	return dispatcherRuntimeStats.stats
}

func recordDispatcherCreated(capacity int) {
	dispatcherRuntimeStats.mu.Lock()
	defer dispatcherRuntimeStats.mu.Unlock()
	dispatcherRuntimeStats.stats.ActiveDispatchers += 1
	dispatcherRuntimeStats.stats.TotalCapacity += capacity
}

func recordDispatcherClosed(capacity int) {
	dispatcherRuntimeStats.mu.Lock()
	defer dispatcherRuntimeStats.mu.Unlock()
	if dispatcherRuntimeStats.stats.ActiveDispatchers > 0 {
		dispatcherRuntimeStats.stats.ActiveDispatchers -= 1
	}
	if dispatcherRuntimeStats.stats.TotalCapacity >= capacity {
		dispatcherRuntimeStats.stats.TotalCapacity -= capacity
	} else {
		dispatcherRuntimeStats.stats.TotalCapacity = 0
	}
}

func recordDispatcherEnqueue(queueDepth int) {
	dispatcherRuntimeStats.mu.Lock()
	defer dispatcherRuntimeStats.mu.Unlock()
	dispatcherRuntimeStats.stats.Enqueued += 1
	dispatcherRuntimeStats.stats.QueueDepth += 1
	if dispatcherRuntimeStats.stats.QueueDepth > dispatcherRuntimeStats.stats.HighWatermark {
		dispatcherRuntimeStats.stats.HighWatermark = dispatcherRuntimeStats.stats.QueueDepth
	}
	if queueDepth > dispatcherRuntimeStats.stats.HighWatermark {
		dispatcherRuntimeStats.stats.HighWatermark = queueDepth
	}
}

func recordDispatcherDropped() {
	dispatcherRuntimeStats.mu.Lock()
	defer dispatcherRuntimeStats.mu.Unlock()
	dispatcherRuntimeStats.stats.Dropped += 1
}

func recordDispatcherProcessed(_ int) {
	dispatcherRuntimeStats.mu.Lock()
	defer dispatcherRuntimeStats.mu.Unlock()
	dispatcherRuntimeStats.stats.Processed += 1
	if dispatcherRuntimeStats.stats.QueueDepth > 0 {
		dispatcherRuntimeStats.stats.QueueDepth -= 1
	}
}

func cloneAlertNode(node *models.Node) models.Node {
	if node == nil {
		return models.Node{}
	}
	return models.Node{
		ID:   node.ID,
		Name: node.Name,
	}
}

func cloneAlertMetrics(metrics *models.MetricsPayload) models.MetricsPayload {
	if metrics == nil {
		return models.MetricsPayload{}
	}

	cloned := models.MetricsPayload{
		TS:  metrics.TS,
		CPU: metrics.CPU,
		Mem: metrics.Mem,
	}
	if len(metrics.Disk) > 0 {
		cloned.Disk = append([]models.DiskStats(nil), metrics.Disk...)
	}
	return cloned
}
