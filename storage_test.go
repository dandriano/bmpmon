package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestStorageBuffer(t *testing.T) {
	storage, err := NewStorage("<connectionstring>", 5)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bufferLimit := 5
	sentLimit := bufferLimit*2 + 1
	responses := make(chan sensorResponse, bufferLimit)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		storage.serve(ctx, responses)
	}()

	sent := 0
	for sent < sentLimit {
		responses <- sensorResponse{
			Altitude:    666,
			Temperature: 18,
			Timestamp:   time.Now(),
		}
		sent += 1

		interval, err := time.ParseDuration("500ms")
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(interval)
	}

	close(responses)
	wg.Wait()

	records, err := storage.Fetch()
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != sentLimit {
		t.Errorf("Expected %d records, got %d", bufferLimit*2+1, len(records))
	}
}
