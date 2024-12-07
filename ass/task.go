package main

import "time"

type Task struct {
	ID          int
	Name        string
	Description string
	Completed   bool
	CompletedAt time.Time
	Interval    Interval
}
