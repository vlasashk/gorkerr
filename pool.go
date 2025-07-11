package gorkerr

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

var (
	ErrNotActive = errors.New("pool is not active")
)

type WorkerPoolWithErr[T any] struct {
	ctx       context.Context
	fn        func(T) error
	jobs      chan T
	eg        *errgroup.Group
	cancel    context.CancelFunc
	workers   int
	once      sync.Once
	isClosed  atomic.Bool
	isStarted atomic.Bool
	inQueue   atomic.Int32
}

func NewWorkerPoolWithErr[T any](workers int, fn func(T) error) *WorkerPoolWithErr[T] {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPoolWithErr[T]{
		workers: workers,
		fn:      fn,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start initiates the worker pool.
// Can only be called once.
// Workers will process jobs until an error occurs or StopAndWait is called.
// When any worker encounters an error, the entire pool shuts down immediately.
func (wp *WorkerPoolWithErr[T]) Start() {
	if !wp.isStarted.CompareAndSwap(false, true) {
		return
	}
	wp.jobs = make(chan T, wp.workers)

	wp.eg, wp.ctx = errgroup.WithContext(wp.ctx)

	for i := 0; i < wp.workers; i++ {
		wp.eg.Go(
			wp.worker(),
		)
	}
}

// shutdown stops worker pool by closing channel, guards WorkerPool.jobs channel to avoid panic.
//
// After calling Close, all incoming payloads from Feed will be ignored to avoid writing in a closed channel
func (wp *WorkerPoolWithErr[T]) shutdown() {
	wp.once.Do(func() {
		// Sets barrier for new incoming payloads
		wp.isClosed.Store(true)
		// Cancels context to free any payload that was blocked on sending to WorkerPool.jobs channel(avoid panic)
		wp.cancel()
		// Waiting to release any payload that might stuck in Feed() select
		for wp.inQueue.Load() > 0 {
			runtime.Gosched()
		}

		close(wp.jobs)
	})
}

// StopAndWait waits until all workers will finish remaining jobs.
func (wp *WorkerPoolWithErr[T]) StopAndWait() error {
	if !wp.isStarted.Load() {
		return ErrNotActive
	}

	wp.shutdown()
	return wp.eg.Wait()
}

// Feed adds a job into queue to perform by workers
func (wp *WorkerPoolWithErr[T]) Feed(payload T) {
	if wp.isClosed.Load() {
		return
	}

	wp.inQueue.Add(1)
	defer wp.inQueue.Add(-1)

	select {
	case <-wp.ctx.Done():
	case wp.jobs <- payload:
	}
}

func (wp *WorkerPoolWithErr[T]) worker() func() error {
	return func() (err error) {
		defer wp.shutdown()

		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("worker panic: %v", r)
			}
		}()

		for job := range wp.jobs {
			if err = wp.fn(job); err != nil {
				return err
			}
		}

		return nil
	}
}
