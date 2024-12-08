package main

import "time"

type Task struct {
	ID          int
	Name        string
	Description string
	CreatedAt   time.Time
	Interval    Interval
}

type TaskResponse struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	CreatedAt          time.Time `json:"created_at"`
	Interval           string    `json:"interval"`
	IntervalsCompleted []int       `json:"intervals_completed"`
}

type Completion struct {
	TaskID      int
	CompletedAt time.Time
}
