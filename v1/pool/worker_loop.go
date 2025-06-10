package pool

import (
	"context"
	"github.com/goriiin/worker-pool/v1/domain"
)

func (wp *WorkerPool) workerLoop(ctx context.Context, w *domain.Worker) {
	defer wp.wg.Done()

	defer func() {
		wp.mu.Lock()
		defer wp.mu.Unlock()

		delete(wp.workers, w.ID)
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case t, ok := <-wp.tasks:
			if !ok {
				return
			}

			wp.execute(t)
		}
	}
}
