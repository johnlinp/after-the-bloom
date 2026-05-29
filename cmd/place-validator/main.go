package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/johnlinp/after-the-bloom/internal/places"
)

const workerCount = 5

type validationJob struct {
	index   int
	placeID string
}

type validationResult struct {
	index int
	place places.ValidatedPlace
	err   error
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	if err := run(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	inputPath := flag.String("input", "", "Input CSV file path")
	outputPath := flag.String("output", "", "Output CSV file path")
	flag.Parse()

	if *inputPath == "" {
		return fmt.Errorf("-input is required")
	}
	if *outputPath == "" {
		*outputPath = defaultValidatorOutputPath(*inputPath)
	}

	client, err := places.NewClient()
	if err != nil {
		return err
	}

	placeIDs, err := places.ReadPlaceIDsCSV(*inputPath)
	if err != nil {
		return err
	}

	log.Printf("validating %d place IDs with %d workers", len(placeIDs), workerCount)

	results := make([]places.ValidatedPlace, len(placeIDs))
	jobs := make(chan validationJob)
	outcomes := make(chan validationResult)

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for job := range jobs {
				details, err := client.GetPlaceDetails(context.Background(), job.placeID)
				if err != nil {
					outcomes <- validationResult{
						index: job.index,
						place: places.ValidatedPlace{
							ID: job.placeID,
						},
						err: err,
					}
					continue
				}

				outcomes <- validationResult{
					index: job.index,
					place: places.ValidatedPlace{
						ID:     job.placeID,
						Name:   details.DisplayName.Text,
						Status: details.BusinessStatus,
					},
				}
			}
		}(i + 1)
	}

	go func() {
		for idx, placeID := range placeIDs {
			jobs <- validationJob{
				index:   idx,
				placeID: placeID,
			}
		}
		close(jobs)
		wg.Wait()
		close(outcomes)
	}()

	var failureCount int
	for outcome := range outcomes {
		if outcome.err != nil {
			failureCount++
			log.Printf("failed to validate %s: %v", outcome.place.ID, outcome.err)
			outcome.place.Status = "ERROR"
		}
		results[outcome.index] = outcome.place
	}

	if err := places.WriteValidatedPlacesCSV(*outputPath, results); err != nil {
		return err
	}

	log.Printf("wrote %d status rows to %s (%d errors)", len(results), *outputPath, failureCount)
	return nil
}

func defaultValidatorOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	if base == "" {
		base = inputPath
	}
	return base + "-statuses.csv"
}
