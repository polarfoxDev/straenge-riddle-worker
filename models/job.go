package models

import "time"

type Job struct {
	Type    string `json:"Type"`
	Payload string `json:"Payload"`
}

type JobResult struct {
	Status     string    `json:"Status"`
	Type       string    `json:"Type"`
	Output     string    `json:"Output"`
	FinishedAt time.Time `json:"FinishedAt"`
}
