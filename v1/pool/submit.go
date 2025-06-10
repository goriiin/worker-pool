package pool

import (
	"context"
	"github.com/goriiin/worker-pool/v1/domain"
)

func (wp *WorkerPool) Submit(ctx context.Context, j domain.Job) (*domain.Future, error) {
	if st := wp.atomicLoadState(); st == ShuttingDown ||
		st == Stopping ||
		st == Stopped {

		return nil, ErrPoolStopped
	}

	f := &domain.Future{
		Result: make(chan domain.FutureResult, 1),
	}

	t := &domain.Task{
		Job:    j,
		Ctx:    ctx,
		Future: f,
	}

	select {
	case wp.tasks <- t:

		return f, nil

	case <-ctx.Done():

		return nil, ctx.Err()
	}
}
