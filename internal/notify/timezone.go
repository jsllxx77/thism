package notify

import (
	"fmt"
	"strings"
	"time"

	"github.com/thism-dev/thism/internal/models"
)

func ValidateTimezoneSettings(settings models.NotificationSettings) error {
	mode := normalizeTimezoneMode(settings.TimeZoneMode)
	if mode != models.NotificationTimeZoneModeCustom {
		return nil
	}

	timeZone := strings.TrimSpace(settings.TimeZone)
	if timeZone == "" {
		return fmt.Errorf("time_zone is required when time_zone_mode is custom")
	}
	if _, err := time.LoadLocation(timeZone); err != nil {
		return fmt.Errorf("invalid notification time_zone")
	}
	return nil
}

func ResolveLocation(settings models.NotificationSettings, systemLocation *time.Location) *time.Location {
	return resolveNotificationLocation(settings, systemLocation)
}

func LocationLabel(location *time.Location, referenceTime time.Time) string {
	return formatNotificationLocationLabel(location, referenceTime)
}

func normalizeTimezoneMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case models.NotificationTimeZoneModeCustom:
		return models.NotificationTimeZoneModeCustom
	default:
		return models.NotificationTimeZoneModeSystem
	}
}

func normalizeSystemLocation(location *time.Location) *time.Location {
	if location != nil {
		return location
	}
	if time.Local != nil {
		return time.Local
	}
	return time.UTC
}

func resolveNotificationLocation(settings models.NotificationSettings, systemLocation *time.Location) *time.Location {
	systemLocation = normalizeSystemLocation(systemLocation)
	if normalizeTimezoneMode(settings.TimeZoneMode) != models.NotificationTimeZoneModeCustom {
		return systemLocation
	}
	if location, err := time.LoadLocation(strings.TrimSpace(settings.TimeZone)); err == nil {
		return location
	}
	return systemLocation
}

func formatNotificationLocationLabel(location *time.Location, referenceTime time.Time) string {
	location = normalizeSystemLocation(location)
	if referenceTime.IsZero() {
		referenceTime = time.Now()
	}
	localTime := referenceTime.In(location)
	_, offsetSeconds := localTime.Zone()
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("%s (UTC%s%02d:%02d)", location.String(), sign, hours, minutes)
}
