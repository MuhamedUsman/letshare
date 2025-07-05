package bgtask

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

var (
	oneBt sync.Once
	bt    *BackgroundTask
)

// BackgroundTask manages a collection of goroutines with a shared lifecycle.
// It provides a mechanism to run, track, and gracefully shut down background tasks.
type BackgroundTask struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	tasks  atomic.Int32
}

// Get returns a singleton BackgroundTask instance.
// It creates the instance on the first call and returns the same instance on subsequent calls.
func Get() *BackgroundTask {
	ctx, cancel := context.WithCancel(context.Background())
	oneBt.Do(func() {
		bt = &BackgroundTask{
			ctx:    ctx,
			cancel: cancel,
		}
	})
	return bt
}

// ShutdownCtx returns the context used by the background task.
// This context will be canceled when shutdown is initiated.
func (bt *BackgroundTask) ShutdownCtx() context.Context {
	return bt.ctx
}

// Run executes the provided function in a goroutine and tracks it.
// The function receives a context that will be canceled when shutdown is initiated.
// Automatically handles panics and decrements task count when the goroutine completes.
func (bt *BackgroundTask) Run(fn func(shutdownCtx context.Context)) {
	bt.wg.Add(1)
	bt.tasks.Add(1)
	go func() {
		defer func() {
			bt.wg.Done()
			bt.tasks.Add(-1)
			if r := recover(); r != nil {
				slog.Error(fmt.Errorf("%v", r).Error())
			}
		}()
		fn(bt.ctx)
	}()
}

func (bt *BackgroundTask) RunAndBlock(fn func(shutdownCtx context.Context)) {
	bt.wg.Add(1)
	bt.tasks.Add(1)
	defer func() {
		bt.wg.Done()
		bt.tasks.Add(-1)
		if r := recover(); r != nil {
			slog.Error(fmt.Errorf("%v", r).Error())
		}
	}()
	fn(bt.ctx)
}

// Shutdown cancels all running tasks and waits for them to complete.
// Returns nil if all tasks complete before the timeout, otherwise returns an error.
// timeout: maximum duration to wait for all tasks to complete.
func (bt *BackgroundTask) Shutdown(timeout time.Duration) error {
	bt.cancel()
	wait := make(chan struct{})
	go func() {
		bt.wg.Wait()
		close(wait)
	}()
	select {
	case <-wait:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout, some background tasks may not have finished, \"count\"=%v", bt.tasks.Load())
	}
}
