package models

import "time"

const TimeLayout = "2006-01-02T15:04:05.000000000Z07:00"

func Now() string { return FormatTime(time.Now()) }

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(TimeLayout)
}

func ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
