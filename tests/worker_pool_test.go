package tests

import (
	"fmt"
	"main/src"
	"main/src/config"
	"sync/atomic"
	"testing"
	"time"
)

type HelperProcessArg struct {
	ID          int
	shouldFail  bool
	sleep       time.Duration
	backChannel chan interface{}
}

func helperProcessTask(args HelperProcessArg) error {
	if args.shouldFail {
		args.backChannel <- nil
		return fmt.Errorf("failed as requested")
	}
	if args.sleep > 0 {
		time.Sleep(args.sleep)
	}
	args.backChannel <- nil
	return nil
}

func TestBasicWorkerPool(t *testing.T) {
	logger := config.NewLogger("TestBasicWorkerPool")
	level, _ := config.ParseLevel("DEBUG")
	logger.SetLevel(level)
	pool := src.NewWorkerPool(10, 100, 2, 5, 5, helperProcessTask, false, logger)
	pool.Start()
	defer pool.Stop()

	backChannel := make(chan interface{}, 10)
	for i := 0; i < 10; i++ {
		arg := HelperProcessArg{
			ID:          i,
			shouldFail:  false,
			sleep:       1,
			backChannel: backChannel,
		}
		pool.Put(arg)
	}

	timeout := time.NewTimer(5 * time.Second)
	// Wait for all tasks to complete
	for i := 0; i < 10; i++ {
		select {
		case <-timeout.C:
			t.Fatalf("Test timed out waiting for tasks to complete")
		case <-backChannel:
		}
	}
}

func TestWorkerPoolPararelization(t *testing.T) {
	logger := config.NewLogger("TestWorkerPoolPararelization")
	level, _ := config.ParseLevel("ERROR")
	logger.SetLevel(level)

	t.Run("testBasics", func(t *testing.T) {
		pool := src.NewWorkerPool(
			10,            // maxConnections
			100,           // maxPending
			1,             // idlePerWorker
			6*time.Second, // workerTimeout
			5,             // baseWorkerCount
			helperProcessTask,
			false,
			logger,
		)
		pool.Start()
		defer pool.Stop()
		time.Sleep(5 * time.Millisecond) // warmup time
		activeWorkers := atomic.LoadInt64(&pool.ActiveWorkers)
		if activeWorkers != 5 { // Needs to be baseWorkerCount
			t.Fatalf("Expected 5 active workers, got %d", activeWorkers)
		}

		backChannel := make(chan interface{}, 10)
		taskTime := 1 * time.Second
		for i := 0; i < 10; i++ {
			pool.Put(HelperProcessArg{i, false, taskTime, backChannel})
		}

		time.Sleep(150 * time.Millisecond)

		activeWorkers = atomic.LoadInt64(&pool.ActiveWorkers)
		if activeWorkers != 5 {
			t.Fatalf("Expected 5 active workers, got %d", activeWorkers)
		}

		timeout := time.NewTimer(2*taskTime + 500*time.Millisecond)
		// Wait for all tasks to complete
		// They should be processed in parallel by 5 workers
		for i := 0; i < 10; i++ {
			select {
			case <-timeout.C:
				t.Fatalf("Test timed out waiting for tasks to complete")
			case <-backChannel:
			}
		}
	})

	t.Run("testNumberOfGoroutines", func(t *testing.T) {
		pool := src.NewWorkerPool(
			10,            // maxConnections
			5,             // maxPending
			0,             // idlePerWorker
			6*time.Second, // workerTimeout
			5,             // baseWorkerCount
			helperProcessTask,
			false,
			logger,
		)
		pool.Start()
		defer pool.Stop()
		warmupTime := 150 * time.Millisecond
		taskTime := 1 * time.Second
		backChannel := make(chan interface{}, 15)
		time.Sleep(warmupTime) // warmup to spawn base workers
		for i := 0; i < 5; i++ {
			pool.Put(HelperProcessArg{i, false, taskTime, backChannel})
		}
		time.Sleep(warmupTime)
		activeWorkers := atomic.LoadInt64(&pool.ActiveWorkers)
		if activeWorkers != 5 { // It should not increase the number of workers
			t.Fatalf("Expected 5 active workers, got %d", activeWorkers)
		}
		for i := 5; i < 10; i++ {
			pool.Put(HelperProcessArg{i, false, taskTime, backChannel})
		}
		time.Sleep(warmupTime)
		activeWorkers = atomic.LoadInt64(&pool.ActiveWorkers)
		if activeWorkers != 10 { // It should increase the number of workers to handle the load
			t.Fatalf("Expected 10 active workers, got %d", activeWorkers)
		}

		for i := 10; i < 15; i++ {
			pool.Put(HelperProcessArg{i, false, taskTime, backChannel})
		}
		time.Sleep(warmupTime)
		activeWorkers = atomic.LoadInt64(&pool.ActiveWorkers)
		if activeWorkers != 10 { // It should not exceed maxConnections
			t.Fatalf("Expected 10 active workers, got %d", activeWorkers)
		}

		shouldErr := pool.Put(HelperProcessArg{16, false, taskTime, backChannel})
		if shouldErr == nil {
			t.Fatalf("Expected error when adding task beyond max pending, got nil")
		}

		timeout := time.NewTimer(4*taskTime + warmupTime)
		// Wait for all tasks to complete
		for i := 0; i < 15; i++ {
			select {
			case <-timeout.C:
				t.Fatalf("Test timed out waiting for tasks to complete")
			case <-backChannel:
			}
		}
	})

	t.Run("testWorkerTimeout", func(t *testing.T) {
		pool := src.NewWorkerPool(
			10,                   // maxConnections
			100,                  // maxPending
			0,                    // idlePerWorker
			500*time.Millisecond, // workerTimeout
			2,                    // baseWorkerCount
			helperProcessTask,
			false,
			logger,
		)
		pool.Start()
		defer pool.Stop()
		time.Sleep(5 * time.Millisecond) // warmup time

		activeWorkers := atomic.LoadInt64(&pool.ActiveWorkers)
		if activeWorkers != 2 { // Needs to be baseWorkerCount
			t.Fatalf("Expected 2 active workers, got %d", activeWorkers)
		}

		backChannel := make(chan interface{}, 5)
		for i := 0; i < 5; i++ {
			pool.Put(HelperProcessArg{i, false, 100 * time.Millisecond, backChannel})
		}

		timeout := time.NewTimer(2 * time.Second)
		// Wait for all tasks to complete
		for i := 0; i < 5; i++ {
			select {
			case <-timeout.C:
				t.Fatalf("Test timed out waiting for tasks to complete")
			case <-backChannel:
			}
		}

		// Wait for longer than worker timeout
		time.Sleep(700 * time.Millisecond)

		activeWorkers = atomic.LoadInt64(&pool.ActiveWorkers)
		if activeWorkers != 2 { // Workers beyond baseWorkerCount should have timed out
			t.Fatalf("Expected 2 active workers after timeout, got %d", activeWorkers)
		}
	})

	t.Run("testFailingTasks", func(t *testing.T) {
		// Pool should not crash on failing tasks
		pool := src.NewWorkerPool(
			10,            // maxConnections
			100,           // maxPending
			2,             // idlePerWorker
			5*time.Second, // workerTimeout
			2,             // baseWorkerCount
			helperProcessTask,
			false,
			logger,
		)
		pool.Start()
		defer pool.Stop()

		backChannel := make(chan interface{}, 5)
		for i := 0; i < 5; i++ {
			pool.Put(HelperProcessArg{i, true, 0, backChannel})
		}

		timeout := time.NewTimer(2 * time.Second)
		// Wait for all tasks to complete
		for i := 0; i < 5; i++ {
			select {
			case <-timeout.C:
				t.Fatalf("Test timed out waiting for tasks to complete")
			case <-backChannel:
			}
		}
	})

}
