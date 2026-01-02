package src

import (
	"fmt"
	"main/src/config"
	"sync/atomic"
	"time"
)

type WorkerPool[T any] struct {
	MaxConnections  int64
	IdlePerWorker   int64
	WorkerTimeout   time.Duration
	BaseWorkerCount int64
	Connections     chan T
	ProcessingFunc  func(T) error
	ActiveWorkers   int64

	FailSilently bool
	Logger       *config.Logger
}

func NewWorkerPool[T any](maxConnections int64, maxPending int64, idlePerWorker int64, workerTimeout time.Duration, baseWorkerCount int64, processingFunc func(T) error, failSilently bool, logger *config.Logger) *WorkerPool[T] {
	return &WorkerPool[T]{
		MaxConnections:  maxConnections,
		IdlePerWorker:   idlePerWorker,
		WorkerTimeout:   workerTimeout,
		BaseWorkerCount: baseWorkerCount,
		Connections:     make(chan T, maxPending),
		ProcessingFunc:  processingFunc,
		FailSilently:    failSilently,
		Logger:          logger,
	}
}

func (c *WorkerPool[T]) Start() {
	for i := int64(0); i < c.BaseWorkerCount; i++ {
		c.spawnWorker(true)
	}
}

func (c *WorkerPool[T]) Stop() {
	close(c.Connections)
}

func (c *WorkerPool[T]) spawnWorker(isBase bool) {
	worker_id := atomic.AddInt64(&c.ActiveWorkers, 1)
	c.Logger.Debug("Spawning new worker #%d. Active workers: %d", worker_id, atomic.LoadInt64(&c.ActiveWorkers))
	go func() {
		defer func() {
			atomic.AddInt64(&c.ActiveWorkers, -1)
			c.Logger.Debug("Worker stopped #%d. Active workers: %d", worker_id, atomic.LoadInt64(&c.ActiveWorkers))
		}()

		var timer *time.Timer
		if !isBase {
			timer = time.NewTimer(c.WorkerTimeout)
			defer timer.Stop()
		}

		for {
			if isBase {
				c.process(<-c.Connections)
			} else {
				select {
				case conn := <-c.Connections:
					// We need to drain the timer channel to avoid spurious timeouts
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					c.Logger.Trace("Worker with id #%d processing", worker_id)
					c.process(conn)
					timer.Reset(c.WorkerTimeout)
				case <-timer.C:
					c.Logger.Debug("Worker with id #%d timed out", worker_id)
					return
				}
			}
		}
	}()
}

func (c *WorkerPool[T]) process(conn T) {
	if err := c.ProcessingFunc(conn); err != nil && !c.FailSilently {
		c.Logger.Error("Error processing in worker: %v", err)
	} else if err != nil {
		c.Logger.Debug("Error processing in worker (silently ignored): %v", err)
	}
}

func (c *WorkerPool[T]) Put(conn T) error {
	select {
	case c.Connections <- conn:
		// Check if we need to spawn a new worker
		active := atomic.LoadInt64(&c.ActiveWorkers)
		pending := int64(len(c.Connections))

		if active < c.MaxConnections && pending > active*c.IdlePerWorker {
			// Try to spawn a new worker if we are not at max capacity
			// We do a double check inside spawn logic usually but here we just fire and forget
			// The worker count is atomic so it's fine.
			// However, we might spawn slightly more than needed if multiple Puts happen at once,
			// but they will timeout eventually.
			// To be strict we should use CAS loop but for this heuristic it might be overkill.
			// So for performance we skip that.
			c.spawnWorker(false)
		}
		return nil
	default:
		c.Logger.Warn("Pool full. Rejected connection.")
		return fmt.Errorf("Pool full")
	}
}
