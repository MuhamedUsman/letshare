package bgtask

import (
	"context"
	"golang.org/x/sync/errgroup"
	"runtime"
)

type task = func() error

type WorkerPool struct {
	Ctx      context.Context
	errGroup *errgroup.Group
}

func NewWorkerPool(ctx context.Context) *WorkerPool {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(4 * runtime.NumCPU())
	return &WorkerPool{
		Ctx:      ctx,
		errGroup: g,
	}
}

func (wp *WorkerPool) Spawn(t task) {
	wp.errGroup.Go(t)
}

func (wp *WorkerPool) Wait() error {
	return wp.errGroup.Wait()
}
