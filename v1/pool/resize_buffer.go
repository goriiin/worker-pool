package pool

import "github.com/goriiin/worker-pool/v1/domain"

func (wp *WorkerPool) ResizeBuffer(size int) error {
	wp.mu.Lock()

	if wp.state != Running {
		wp.mu.Unlock()

		return ErrNotRunning
	}

	if size < 0 {
		wp.mu.Unlock()

		return ErrInvalidSize
	}

	if size == wp.jobsBufferSize {
		wp.mu.Unlock()

		return nil
	}

	oldTasks := wp.tasks
	newTasks := make(chan *domain.Task, size)
	wp.tasks = newTasks
	wp.jobsBufferSize = size

	wp.mu.Unlock()

	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		close(oldTasks)

		for task := range oldTasks {
			select {
			case newTasks <- task:
			case <-wp.ctx.Done():
				task.Future.Result <- domain.FutureResult{Err: ErrPoolStopped}
				close(task.Future.Result)

				for remainingTask := range oldTasks {
					remainingTask.Future.Result <- domain.FutureResult{Err: ErrPoolStopped}
					close(remainingTask.Future.Result)
				}

				return
			default:
				task.Future.Result <- domain.FutureResult{Err: ErrPoolStopped}
				close(task.Future.Result)
			}
		}
	}()

	return nil
}
