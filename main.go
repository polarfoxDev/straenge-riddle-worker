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
	client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
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

	logrus.Info("Started worker...")

	for {
		logrus.Info("Waiting for jobs...")
		jobRaw, err := client.RPopLPush(ctx, "tasks", "processing").Result()
		if err == redis.Nil {
			time.Sleep(5 * time.Second)
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

		ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(jobTimeout)*time.Second)

		riddle, err := processJob(ctxTimeout, job, parallelCount)

		cancel()

		client.LRem(ctx, "processing", 1, jobRaw) // clean up processing list

		if err != nil {
			logrus.Errorf("❌ Job failed: %v", err)
			client.LPush(ctx, "retry", jobRaw)
			continue
		}

		output := convert.TransformToOutputFormat(riddle, job.Type)

		outputJson, err := json.Marshal(output)
		if err != nil {
			logrus.Errorf("❌ Output could not be serialized: %v", err)
			client.LPush(ctx, "retry", jobRaw)
			continue
		}

		res := models.JobResult{
			Status:     "done",
			Type:       job.Type,
			Output:     string(outputJson),
			FinishedAt: time.Now().UTC(),
		}
		resJson, err := json.Marshal(res) // handle error
		if err != nil {
			logrus.Errorf("❌ Result could not be serialized: %v", err)
			client.LPush(ctx, "retry", jobRaw)
			continue
		}
		client.LPush(ctx, "results", resJson)
		logrus.Info("✅ Job successfully processed and result saved.")

		time.Sleep(30 * time.Second)
	}
}
