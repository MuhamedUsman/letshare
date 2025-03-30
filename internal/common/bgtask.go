package common

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var (
	bt   *BackgroundTask
	once sync.Once
)

type BackgroundTask struct {
	wg     *sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	Tasks  int
}

func NewBackgroundTask() *BackgroundTask {
	ctx, cancel := context.WithCancel(context.Background())
	once.Do(func() {
		bt = &BackgroundTask{
			wg:     &sync.WaitGroup{},
			ctx:    ctx,
			cancel: cancel,
		}
	})
	return bt
}

func (bt *BackgroundTask) Run(fn func(shutdownCtx context.Context)) {
	bt.wg.Add(1)
	bt.Tasks++
	go func() {
		defer func() {
			bt.wg.Done()
			bt.Tasks--
			if r := recover(); r != nil {
				slog.Error(fmt.Errorf("%v", r).Error())
			}
		}()
		fn(bt.ctx)
	}()
}

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
		return fmt.Errorf("shutdown timeout, some background tasks may not have finished, \"count\"=%v", bt.Tasks)
	}
}
