package main

import "strings"

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
