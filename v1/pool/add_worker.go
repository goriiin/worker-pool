package pool

import (
	"context"
	"github.com/goriiin/worker-pool/v1/domain"
	"sync/atomic"
)

func (wp *WorkerPool) addWorker() {
	workerID := atomic.AddUint64(&wp.nextID, 1)

	ctx, cancel := context.WithCancel(wp.ctx)
	w := &domain.Worker{ID: workerID, Cancel: cancel}

	wp.mu.Lock()
	wp.workers[w.ID] = w
	wp.wg.Add(1)
	wp.mu.Unlock()

	go wp.workerLoop(ctx, w)
}
