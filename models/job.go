package models

import "time"

type Job struct {
	Type    string `json:"Type"`
	Payload string `json:"Payload"`
}

type JobSuccess struct {
	SuperSolution string    `json:"SuperSolution"`
	Output        string    `json:"Output"`
	StartedAt     time.Time `json:"StartedAt"`
	FinishedAt    time.Time `json:"FinishedAt"`
	ParallelCount int       `json:"ParallelCount"`
}
