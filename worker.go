package main

import (
	"context"
	"encoding/json"
	"fmt"
	"straenge-riddle-worker/m/models"
	"sync"

	"github.com/sirupsen/logrus"
)

func processJob(ctx context.Context, job models.Job, parallelCount int) (*models.Riddle, error) {
	logrus.Infof("Processing Job: %s with payload: %s", job.Type, job.Payload)

	var riddleConcept *models.RiddleConcept

	// Try parsing as JobPayload (new format with jobId)
	var jobPayload models.JobPayload
	if err := json.Unmarshal([]byte(job.Payload), &jobPayload); err == nil && jobPayload.JobID != "" {
		riddleConcept = &jobPayload.RiddleConcept
		logrus.Infof("Parsed job with ID: %s", jobPayload.JobID)
	} else {
		// Fall back to legacy format (RiddleConcept directly)
		var legacyConcept models.RiddleConcept
		if err := json.Unmarshal([]byte(job.Payload), &legacyConcept); err != nil {
			return nil, fmt.Errorf("error processing job: %v", err)
		}
		riddleConcept = &legacyConcept
	}

	superSolution := riddleConcept.SuperSolution
	wordPool := riddleConcept.WordPool
	logrus.Infof("Processing riddle with super solution: %s and %d words", superSolution, len(wordPool))

	riddle := generateRiddle(ctx, superSolution, wordPool, parallelCount)
	if riddle == nil {
		logrus.Warn("Failed to generate riddle")
		return nil, fmt.Errorf("failed to generate riddle")
	}
	return riddle, nil
}

func generateRiddleSingleTry(superSolution string, wordPool []string) *models.Riddle {
	logrus.Infof("Running riddle generation for super solution: %s", superSolution)
	var riddle, err = models.NewRiddle(superSolution, wordPool)
	if err != nil {
		logrus.Warn("Failed to create starting riddle")
		logrus.Warn(err)
		return nil
	}
	riddle.Render(true)
	riddle, err = riddle.FillWithWords()
	if err != nil {
		logrus.Warn("Failed to fill riddle with words")
		logrus.Warn(err)
		return nil
	}
	var ambiguous, _ = riddle.CheckForAmbiguity()
	if ambiguous {
		logrus.Warn("Generated riddle is ambiguous")
		return nil
	}
	logrus.Info("Riddle generation successful")
	return riddle
}

func tryRiddleGenerationInParallel(superSolution string, wordPool []string, parallelCount int) *models.Riddle {
	if parallelCount <= 1 {
		logrus.Info("Parallel count is 1 or less, running single generation")
		return generateRiddleSingleTry(superSolution, wordPool)
	}

	logrus.Infof("Starting riddle generation in parallel with %d goroutines", parallelCount)

	var wg sync.WaitGroup
	resultChan := make(chan *models.Riddle, 1)

	for i := 0; i < parallelCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			res := generateRiddleSingleTry(superSolution, wordPool)
			if res != nil {
				select {
				case resultChan <- res:
				default:
				}
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	if result, ok := <-resultChan; ok {
		return result
	}

	return nil
}

func generateRiddle(ctx context.Context, superSolution string, wordPool []string, parallelCount int) *models.Riddle {
	for i := 0; ; i++ {
		if ctx.Err() != nil {
			logrus.Warn("Reached Timeout, stopping riddle generation")
			return nil
		}
		riddle := tryRiddleGenerationInParallel(superSolution, wordPool, parallelCount)
		if riddle != nil {
			return riddle
		}
		logrus.Info("Retrying riddle generation...")
	}
}
