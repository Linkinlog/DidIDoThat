package main

import (
	"strings"
	"time"
)

func fromString(s string) Interval {
	s = strings.ToLower(s)
	switch s {
	case "hourly":
		return Hourly
	case "daily":
		return Daily
	case "weekly":
		return Weekly
	case "monthly":
		return Monthly
	default:
		return 0
	}
}

type Interval int

const (
	_ Interval = iota
	Hourly
	Daily
	Weekly
	Monthly
)

func (i Interval) String() string {
	return [...]string{"", "Hourly", "Daily", "Weekly", "Monthly"}[i]
}

func (i Interval) toTime() time.Duration {
	switch i {
	case Hourly:
		return time.Hour
	case Daily:
		return 24 * time.Hour
	case Weekly:
		return 7 * 24 * time.Hour
	case Monthly:
		return 24 * 30 * time.Hour
	default:
		return 0
	}
}
