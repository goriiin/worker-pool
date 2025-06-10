package pool

import (
	"github.com/goriiin/worker-pool/v1/domain"
	"log"
)

func (wp *WorkerPool) Start(poolSize int) error {
	log.Println("[ DEBUG ] WorkerPool.Start started")

	if poolSize < 0 {
		poolSize = 0
	}

	wp.mu.Lock()

	if wp.state != Created {
		wp.mu.Unlock()

		return ErrPoolAlreadyRunning
	}

	wp.atomicStoreState(Running)
	wp.tasks = make(chan *domain.Task, wp.jobsBufferSize)

	wp.mu.Unlock()

	for i := 0; i < poolSize; i++ {
		wp.addWorker()
	}

	log.Println("[ DEBUG ] WorkerPool.Start successful ends")

	return nil
}
