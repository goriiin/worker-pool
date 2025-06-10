package pool

import "sync/atomic"

func (wp *WorkerPool) atomicStoreState(state State) {
	atomic.StoreInt32((*int32)(&wp.state), int32(state))
}
