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

func init() {
	// read dotenv file
	err := godotenv.Load()
	if err != nil {
		logrus.Warn("No .env file found")
	}
	// setup logging
	lvl, ok := os.LookupEnv("LOG_LEVEL")
	// LOG_LEVEL not set, let's default to info
	if !ok {
		lvl = "info"
	}
	// parse string, this is built-in feature of logrus
	ll, err := logrus.ParseLevel(lvl)
	if err != nil {
		ll = logrus.InfoLevel
	}
	// set global log level
	logrus.SetLevel(ll)
	logrus.Info("Logging initialized with level: ", lvl)
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
			logrus.Errorf("❌ Invalid Job: %v", err)
			client.LRem(ctx, "processing", 1, jobRaw)
			continue
		}

		timeout := jobTimeout
		if job.Type == "retry" {
			timeout = retryTimeout
		}

		startedAt := time.Now().UTC()

		logrus.Infof("Job type: %s, timeout: %d seconds", job.Type, timeout)
		ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)

		riddle, err := processJob(ctxTimeout, job, parallelCount)

		cancel()

		client.LRem(ctx, "processing", 1, jobRaw) // clean up processing list

		if err != nil {
			logrus.Errorf("❌ Job failed: %v", err)
			if job.Type != "retry" {
				job.Type = "retry"
				jobRaw, err := json.Marshal(job)
				if err != nil {
					continue
				}
				client.LPush(ctx, "generate-riddle", jobRaw)
			}
			continue
		}

		var riddleConcept models.RiddleConcept
		err = json.Unmarshal([]byte(job.Payload), &riddleConcept)
		if err != nil {
			logrus.Errorf("❌ Riddle concept could not be deserialized: %v", err)
			continue
		}

		output := convert.TransformToOutputFormat(riddle, riddleConcept.ThemeDescription)

		outputJson, err := json.Marshal(output)
		if err != nil {
			logrus.Errorf("❌ Output could not be serialized: %v", err)
			if job.Type != "retry" {
				job.Type = "retry"
				jobRaw, err := json.Marshal(job)
				if err != nil {
					continue
				}
				client.LPush(ctx, "generate-riddle", jobRaw)
			}
			continue
		}

		res := models.JobSuccess{
			ParallelCount: parallelCount,
			SuperSolution: riddleConcept.SuperSolution,
			Output:        string(outputJson),
			StartedAt:     startedAt,
			FinishedAt:    time.Now().UTC(),
		}
		resJson, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("❌ Result could not be serialized: %v", err)
			if job.Type != "retry" {
				job.Type = "retry"
				jobRaw, err := json.Marshal(job)
				if err != nil {
					continue
				}
				client.LPush(ctx, "generate-riddle", jobRaw)
			}
			continue
		}
		client.LPush(ctx, "generate-riddle-result", resJson)
		logrus.Info("✅ Job successfully processed and result saved to queue generate-riddle-result")
	}
}
