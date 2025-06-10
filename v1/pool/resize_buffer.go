package pool

import "github.com/goriiin/worker-pool/v1/domain"

func (wp *WorkerPool) ResizeBuffer(size int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.state != Running {
		return ErrNotRunning
	}

	if size < 0 {
		return ErrInvalidSize
	}

	if size == wp.jobsBufferSize {
		return nil
	}

	oldTasks := wp.tasks

	newTasks := make(chan *domain.Task, size)
	wp.tasks = newTasks
	wp.jobsBufferSize = size

	go func() {
		defer func() {
			if r := recover(); r != nil {
				for t := range oldTasks {
					t.Future.Result <- domain.FutureResult{Err: ErrPoolStopped}
					close(t.Future.Result)
				}
			}
		}()

		close(oldTasks)

		for t := range oldTasks {
			newTasks <- t
		}
	}()

	return nil
}
