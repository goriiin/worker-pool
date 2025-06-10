package pool

import "context"

func (wp *WorkerPool) Resize(size int) error {
	wp.mu.Lock()

	if wp.state != Running {
		wp.mu.Unlock()

		return ErrNotRunning
	}

	if size < 0 {
		wp.mu.Unlock()

		return ErrInvalidSize
	}

	curr := len(wp.workers)
	diff := size - curr

	switch {
	case diff > 0:
		wp.mu.Unlock()

		for i := 0; i < diff; i++ {
			go wp.addWorker()
		}

		return nil
	case diff < 0:
		diff = -diff
		i := 0
		workersToCancel := make([]context.CancelFunc, 0, diff)

		for _, w := range wp.workers {
			if i >= diff {
				break
			}

			workersToCancel = append(workersToCancel, w.Cancel)
			delete(wp.workers, w.ID)

			i++
		}
		wp.mu.Unlock()

		for _, cancel := range workersToCancel {
			cancel()
		}

		return nil

	default:
		wp.mu.Unlock()

		return nil
	}
}
