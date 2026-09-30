package models

import "time"

const TimeLayout = time.RFC3339

func Now() string { return FormatTime(time.Now()) }

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(TimeLayout)
}

func ParseTime(s string) (time.Time, error) {
	return time.Parse(TimeLayout, s)
}
