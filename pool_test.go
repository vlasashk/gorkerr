package gorkerr_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vlasashk/gorkerr"
)

func TestSimpleProcessing(t *testing.T) {
	var count int32
	wp := gorkerr.NewWorkerPoolWithErr[int](5, func(i int) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	wp.Start()

	totalJobs := 100
	for i := 0; i < totalJobs; i++ {
		wp.Feed(i)
	}

	err := wp.StopAndWait()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if atomic.LoadInt32(&count) != int32(totalJobs) {
		t.Fatalf("Expected count %d, got %d", totalJobs, count)
	}
}

func TestErrorHandling(t *testing.T) {
	var count int32
	targetErr := errors.New("test error")
	wp := gorkerr.NewWorkerPoolWithErr[int](5, func(i int) error {
		if i == 50 {
			return targetErr
		}
		atomic.AddInt32(&count, 1)
		return nil
	})

	wp.Start()

	totalJobs := 100
	for i := 0; i < totalJobs; i++ {
		wp.Feed(i)
	}

	err := wp.StopAndWait()
	if !errors.Is(err, targetErr) {
		t.Fatalf("Expected error %v, got %v", targetErr, err)
	}

	// Count may be greater than 50 due to concurrency
	c := atomic.LoadInt32(&count)
	if c < 50 || c > int32(totalJobs) {
		t.Fatalf("Unexpected count %d", c)
	}
}

func TestConcurrentFeed(t *testing.T) {
	var count int32
	wp := gorkerr.NewWorkerPoolWithErr[int](5, func(i int) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	wp.Start()

	totalJobs := 100
	numFeeders := 10
	var wg sync.WaitGroup
	wg.Add(numFeeders)
	for f := 0; f < numFeeders; f++ {
		go func() {
			defer wg.Done()
			for i := 0; i < totalJobs/numFeeders; i++ {
				wp.Feed(i)
			}
		}()
	}

	wg.Wait()
	err := wp.StopAndWait()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if atomic.LoadInt32(&count) != int32(totalJobs) {
		t.Fatalf("Expected count %d, got %d", totalJobs, count)
	}
}

func TestFeedAfterStop(t *testing.T) {
	var count int32
	wp := gorkerr.NewWorkerPoolWithErr[int](5, func(i int) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	wp.Start()
	wp.Feed(1)
	err := wp.StopAndWait()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Feeding after StopAndWait should have no effect
	wp.Feed(2)
	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("Expected count 1, got %d", count)
	}
}

func TestStartMultipleTimes(t *testing.T) {
	var count int32
	wp := gorkerr.NewWorkerPoolWithErr[int](5, func(i int) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	wp.Start()
	wp.Start() // Should have no effect

	wp.Feed(1)
	err := wp.StopAndWait()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("Expected count 1, got %d", count)
	}
}

func TestWorkerPanicHandling(t *testing.T) {
	wp := gorkerr.NewWorkerPoolWithErr[int](5, func(i int) error {
		panic("test panic")
	})

	wp.Start()
	wp.Feed(1)
	err := wp.StopAndWait()
	expectedErr := fmt.Errorf("worker panic: %v", "test panic")
	if err == nil || err.Error() != expectedErr.Error() {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestStopWithoutStart(t *testing.T) {
	wp := gorkerr.NewWorkerPoolWithErr[int](5, func(i int) error {
		return nil
	})

	err := wp.StopAndWait()
	if !errors.Is(err, gorkerr.ErrNotActive) {
		t.Fatalf("Expected error %v, got %v", gorkerr.ErrNotActive, err)
	}
}

func TestStopMultipleTimes(t *testing.T) {
	var count int32
	wp := gorkerr.NewWorkerPoolWithErr[int](5, func(i int) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	wp.Start()
	wp.Feed(1)
	err := wp.StopAndWait()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Subsequent StopAndWait calls should return ErrNotActive
	err = wp.StopAndWait()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestFeedWhileProcessing(t *testing.T) {
	var count int32
	wp := gorkerr.NewWorkerPoolWithErr[int](2, func(i int) error {
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&count, 1)
		return nil
	})

	wp.Start()
	totalJobs := 10
	for i := 0; i < totalJobs; i++ {
		wp.Feed(i)
	}
	err := wp.StopAndWait()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if atomic.LoadInt32(&count) != int32(totalJobs) {
		t.Fatalf("Expected count %d, got %d", totalJobs, count)
	}
}

func TestStopProcessingWithActiveFeed(t *testing.T) {
	var count int32
	workers := int32(5)
	wp := gorkerr.NewWorkerPoolWithErr[int](int(workers), func(i int) error {
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&count, 1)
		return nil
	})

	wp.Start()
	totalJobs := 20

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < totalJobs; i++ {
			wp.Feed(i)
		}
	}()

	time.Sleep(1 * time.Millisecond)
	err := wp.StopAndWait()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	wg.Wait()

	// workers*2 since first half will be immediately assigned to workers and second half will be waiting in queue
	if atomic.LoadInt32(&count) != workers*2 {
		t.Fatalf("Expected count %d, got %d", totalJobs, count)
	}
}
