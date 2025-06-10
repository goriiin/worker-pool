package pool

import (
	"github.com/goriiin/worker-pool/v1/domain"
	"sync/atomic"
)

func (wp *WorkerPool) Stop() {
	wp.mu.Lock()

	if wp.state == Stopped || wp.state == Stopping {
		wp.mu.Unlock()

		return
	}

	atomic.StoreInt32((*int32)(&wp.state), int32(Stopping))

	wp.cancel()
	close(wp.tasks)

	wp.mu.Unlock()
	wp.wg.Wait()

	wp.mu.Lock()
	for t := range wp.tasks {
		t.Future.Result <- domain.FutureResult{Err: ErrPoolStopped}
		close(t.Future.Result)
	}
	wp.mu.Unlock()

	wp.atomicStoreState(Stopped)
}
