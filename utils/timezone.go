package utils

import (
	"time"
)

var (
	// IST timezone location
	ISTLocation *time.Location
)

// InitTimezone initializes the IST timezone
func InitTimezone() {
	var err error
	ISTLocation, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		// Fallback to fixed offset if timezone loading fails
		ISTLocation = time.FixedZone("IST", 5*60*60+30*60) // UTC+5:30
	}
}

// GetISTTime returns current time in IST
func GetISTTime() time.Time {
	return time.Now().In(ISTLocation)
}

// ConvertToIST converts any time to IST
func ConvertToIST(t time.Time) time.Time {
	return t.In(ISTLocation)
}

// FormatISTTime formats time in IST with standard format
func FormatISTTime(t time.Time) string {
	return t.In(ISTLocation).Format("2006-01-02 15:04:05")
}

// FormatISTTimeCustom formats time in IST with custom format
func FormatISTTimeCustom(t time.Time, layout string) string {
	return t.In(ISTLocation).Format(layout)
}

// ParseTimeInIST parses time string and returns time in IST
func ParseTimeInIST(layout, value string) (time.Time, error) {
	t, err := time.ParseInLocation(layout, value, ISTLocation)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// GetISTTimeString returns current IST time as formatted string
func GetISTTimeString() string {
	return GetISTTime().Format("2006-01-02 15:04:05")
}

// GetISTDateString returns current IST date as formatted string
func GetISTDateString() string {
	return GetISTTime().Format("2006-01-02")
}

// IsWithinBusinessHours checks if current IST time is within business hours
func IsWithinBusinessHours(openTime, closeTime string) bool {
	now := GetISTTime()

	open, err := time.Parse("15:04:05", openTime)
	if err != nil {
		return false
	}

	close, err := time.Parse("15:04:05", closeTime)
	if err != nil {
		return false
	}

	openDateTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		open.Hour(), open.Minute(), open.Second(), 0, ISTLocation,
	)

	closeDateTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		close.Hour(), close.Minute(), close.Second(), 0, ISTLocation,
	)

	// Handle overnight businesses (e.g. 18:00 - 02:00)
	if closeDateTime.Before(openDateTime) {
		closeDateTime = closeDateTime.Add(24 * time.Hour)
	}

	return now.After(openDateTime) && now.Before(closeDateTime)
}
