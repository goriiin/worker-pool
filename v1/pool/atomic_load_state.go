package pool

import "sync/atomic"

func (wp *WorkerPool) atomicLoadState() State {
	return State(atomic.LoadInt32((*int32)(&wp.state)))
}
