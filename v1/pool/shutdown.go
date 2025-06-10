package pool

func (wp *WorkerPool) Shutdown() {
	wp.mu.Lock()

	if wp.state == ShuttingDown ||
		wp.state == Stopped ||
		wp.state == Stopping {
		wp.mu.Unlock()

		return
	}

	wp.atomicStoreState(ShuttingDown)

	close(wp.tasks)
	wp.mu.Unlock()

	wp.wg.Wait()

	wp.atomicStoreState(Stopped)
}
