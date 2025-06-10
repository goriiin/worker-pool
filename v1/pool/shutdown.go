package pool

import "log"

func (wp *WorkerPool) Shutdown() {
	log.Println("Shutting down workers...")

	if st := wp.atomicLoadState(); st == ShuttingDown ||
		st == Stopped ||
		st == Stopping {
		return
	}

	wp.atomicStoreState(ShuttingDown)

	wp.mu.Lock()

	for _, w := range wp.workers {
		w.Cancel()
	}

	wp.mu.Unlock()

	wp.wg.Wait()

	close(wp.tasks)

	wp.atomicStoreState(Stopped)

	log.Println("Shutting down success")
}
