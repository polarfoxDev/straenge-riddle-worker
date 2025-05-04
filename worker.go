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
	logrus.Infof("🛠 Processing Job: %s with payload: %s\n", job.Type, job.Payload)
	// extract riddle concept from job payload
	var riddleConcept models.RiddleConcept
	err := json.Unmarshal([]byte(job.Payload), &riddleConcept)
	if err != nil {
		return nil, fmt.Errorf("error processing job: %v", err)
	}

	superSolution := riddleConcept.SuperSolution
	wordPool := riddleConcept.WordPool
	riddle := generateRiddle(ctx, superSolution, wordPool, parallelCount)
	if riddle == nil {
		logrus.Warn("Failed to generate riddle")
		return nil, fmt.Errorf("failed to generate riddle")
	}
	return riddle, nil
}

func generateRiddleSingleTry(superSolution string, wordPool []string) *models.Riddle {
	logrus.Info("Running riddle generation...")
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

	// Function to run in parallel
	for i := 0; i < parallelCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			res := generateRiddleSingleTry(superSolution, wordPool)
			if res != nil {
				select {
				case resultChan <- res:
					// Signal found result
				default:
					// Ignore if result already sent
				}
			}
		}(i)
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Wait for the first non-nil result or completion
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
