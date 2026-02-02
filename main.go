package main

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"straenge-riddle-worker/m/convert"
	"straenge-riddle-worker/m/models"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var (
	ctx    = context.Background()
	client *redis.Client
)

// JobMetadata mirrors the API's job metadata structure
type JobMetadata struct {
	JobID         string             `json:"jobId"`
	Status        string             `json:"status"`
	RiddleConcept models.RiddleConcept `json:"riddleConcept"`
	CreatedAt     time.Time          `json:"createdAt"`
	StartedAt     *time.Time         `json:"startedAt,omitempty"`
	FinishedAt    *time.Time         `json:"finishedAt,omitempty"`
	Error         string             `json:"error,omitempty"`
	Result        string             `json:"result,omitempty"`
}

func init() {
	godotenv.Load()
	logLevel, success := os.LookupEnv("LOG_LEVEL")
	if !success {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			logrus.SetLevel(logrus.InfoLevel)
		} else {
			logrus.SetLevel(level)
		}
	}
}

// updateJobStatus updates the status of a job in Redis
func updateJobStatus(jobID string, status string, startedAt *time.Time, finishedAt *time.Time, errorMsg string, result string) error {
	if jobID == "" {
		return nil // Legacy job without ID
	}

	key := "job:" + jobID
	existing, err := client.Get(ctx, key).Result()
	if err != nil {
		logrus.Warnf("Could not get existing job metadata for %s: %v", jobID, err)
		return err
	}

	var metadata JobMetadata
	if err := json.Unmarshal([]byte(existing), &metadata); err != nil {
		logrus.Warnf("Could not unmarshal job metadata for %s: %v", jobID, err)
		return err
	}

	metadata.Status = status
	if startedAt != nil {
		metadata.StartedAt = startedAt
	}
	if finishedAt != nil {
		metadata.FinishedAt = finishedAt
	}
	if errorMsg != "" {
		metadata.Error = errorMsg
	}
	if result != "" {
		metadata.Result = result
	}

	updated, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return client.Set(ctx, key, updated, 24*time.Hour).Err()
}

// parseJobPayload extracts job ID and riddle concept from the payload
func parseJobPayload(payload string) (string, *models.RiddleConcept, error) {
	// Try new format first
	var jobPayload models.JobPayload
	if err := json.Unmarshal([]byte(payload), &jobPayload); err == nil && jobPayload.JobID != "" {
		return jobPayload.JobID, &jobPayload.RiddleConcept, nil
	}

	// Fall back to legacy format
	var riddleConcept models.RiddleConcept
	if err := json.Unmarshal([]byte(payload), &riddleConcept); err != nil {
		return "", nil, err
	}
	return "", &riddleConcept, nil
}

func main() {
	redisUrl, success := os.LookupEnv("REDIS_URL")
	if !success {
		logrus.Fatal("REDIS_URL not set")
		return
	}
	client = redis.NewClient(&redis.Options{
		Addr: redisUrl,
	})

	parallelCountStr, success := os.LookupEnv("PARALLEL_COUNT")
	if !success {
		logrus.Fatal("PARALLEL_COUNT not set")
		return
	}
	parallelCount, err := strconv.Atoi(parallelCountStr)
	if err != nil {
		logrus.Fatalf("Invalid PARALLEL_COUNT value: %v", err)
		return
	}

	jobTimeoutStr, success := os.LookupEnv("JOB_TIMEOUT_SECONDS")
	if !success {
		logrus.Fatal("JOB_TIMEOUT_SECONDS not set")
		return
	}
	jobTimeout, err := strconv.Atoi(jobTimeoutStr)
	if err != nil {
		logrus.Fatalf("Invalid JOB_TIMEOUT_SECONDS value: %v", err)
		return
	}

	retryTimeoutStr, success := os.LookupEnv("RETRY_TIMEOUT_SECONDS")
	if !success {
		logrus.Fatal("RETRY_TIMEOUT_SECONDS not set")
		return
	}
	retryTimeout, err := strconv.Atoi(retryTimeoutStr)
	if err != nil {
		logrus.Fatalf("Invalid RETRY_TIMEOUT_SECONDS value: %v", err)
		return
	}

	logrus.Info("Started worker...")

	for {
		time.Sleep(10 * time.Second)
		logrus.Info("Waiting for jobs of source generate-riddle...")
		jobRaw, err := client.RPopLPush(ctx, "generate-riddle", "processing").Result()
		if err == redis.Nil {
			continue
		} else if err != nil {
			logrus.Errorf("Redis Error: %v", err)
			continue
		}

		var job models.Job
		if err := json.Unmarshal([]byte(jobRaw), &job); err != nil {
			logrus.Errorf("Invalid Job: %v", err)
			client.LRem(ctx, "processing", 1, jobRaw)
			continue
		}

		// Parse job payload to get job ID and riddle concept
		jobID, riddleConcept, err := parseJobPayload(job.Payload)
		if err != nil {
			logrus.Errorf("Failed to parse job payload: %v", err)
			client.LRem(ctx, "processing", 1, jobRaw)
			continue
		}

		timeout := jobTimeout
		if job.Type == "retry" {
			timeout = retryTimeout
		}

		startedAt := time.Now().UTC()

		// Update job status to "working"
		if err := updateJobStatus(jobID, "working", &startedAt, nil, "", ""); err != nil {
			logrus.Warnf("Failed to update job status to working: %v", err)
		}

		logrus.Infof("Job ID: %s, type: %s, timeout: %d seconds", jobID, job.Type, timeout)
		ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)

		riddle, err := processJob(ctxTimeout, job, parallelCount)

		cancel()

		client.LRem(ctx, "processing", 1, jobRaw) // clean up processing list

		if err != nil {
			logrus.Errorf("Job failed: %v", err)
			finishedAt := time.Now().UTC()
			if job.Type != "retry" {
				// Update status to queued for retry
				if updateErr := updateJobStatus(jobID, "queued", nil, nil, "", ""); updateErr != nil {
					logrus.Warnf("Failed to update job status: %v", updateErr)
				}
				job.Type = "retry"
				jobRaw, err := json.Marshal(job)
				if err != nil {
					continue
				}
				client.LPush(ctx, "generate-riddle", jobRaw)
			} else {
				// Update status to failed
				if updateErr := updateJobStatus(jobID, "failed", nil, &finishedAt, err.Error(), ""); updateErr != nil {
					logrus.Warnf("Failed to update job status: %v", updateErr)
				}
			}
			continue
		}

		output := convert.TransformToOutputFormat(riddle, riddleConcept.ThemeDescription)

		outputJson, err := json.Marshal(output)
		if err != nil {
			logrus.Errorf("Output could not be serialized: %v", err)
			finishedAt := time.Now().UTC()
			if updateErr := updateJobStatus(jobID, "failed", nil, &finishedAt, "Output serialization failed", ""); updateErr != nil {
				logrus.Warnf("Failed to update job status: %v", updateErr)
			}
			continue
		}

		finishedAt := time.Now().UTC()

		// Update job status to "completed"
		if err := updateJobStatus(jobID, "completed", nil, &finishedAt, "", string(outputJson)); err != nil {
			logrus.Warnf("Failed to update job status to completed: %v", err)
		}

		res := models.JobSuccess{
			JobID:         jobID,
			ParallelCount: parallelCount,
			SuperSolution: riddleConcept.SuperSolution,
			Output:        string(outputJson),
			StartedAt:     startedAt,
			FinishedAt:    finishedAt,
		}
		resJson, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("Result could not be serialized: %v", err)
			continue
		}
		client.LPush(ctx, "generate-riddle-result", resJson)
		logrus.Info("Job successfully processed and result saved to queue generate-riddle-result")
	}
}

