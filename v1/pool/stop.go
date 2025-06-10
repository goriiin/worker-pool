package pool

import (
	"github.com/goriiin/worker-pool/v1/domain"
)

func (wp *WorkerPool) Stop() {
	if st := wp.atomicLoadState(); st == Stopped ||
		st == Stopping {
		return
	}

	wp.mu.Lock()

	if wp.state == Stopped ||
		wp.state == Stopping {
		wp.mu.Unlock()

		return
	}

	wp.atomicStoreState(Stopping)

	for _, w := range wp.workers {
		w.Cancel()
	}

	close(wp.tasks)

	remainingTasks := make([]*domain.Task, 0, len(wp.tasks))
	for task := range wp.tasks {
		remainingTasks = append(remainingTasks, task)
	}

	for t := range wp.tasks {
		t.Future.Result <- domain.FutureResult{Err: ErrPoolStopped}
		close(t.Future.Result)
	}

	wp.mu.Unlock()

	wp.cancel()
	wp.wg.Wait()

	wp.atomicStoreState(Stopped)
}
