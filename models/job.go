package models

import "time"

type Job struct {
	Type    string `json:"Type"`
	Payload string `json:"Payload"`
}

// JobPayload is the payload structure from the API that includes a job ID
type JobPayload struct {
	JobID         string        `json:"jobId"`
	RiddleConcept RiddleConcept `json:"riddleConcept"`
}

type JobSuccess struct {
	JobID         string    `json:"jobId,omitempty"`
	SuperSolution string    `json:"SuperSolution"`
	Output        string    `json:"Output"`
	StartedAt     time.Time `json:"StartedAt"`
	FinishedAt    time.Time `json:"FinishedAt"`
	ParallelCount int       `json:"ParallelCount"`
}

