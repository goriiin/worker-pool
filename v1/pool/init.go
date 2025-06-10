package pool

import (
	"context"
	"errors"
	"github.com/goriiin/worker-pool/v1/domain"
	"sync"
)

var (
	ErrPoolStopped        = errors.New("worker pool is stopped or stopping")
	ErrPoolAlreadyRunning = errors.New("worker pool is already running")
	ErrJobPanic           = errors.New("worker pool job panic")
	ErrInvalidSize        = errors.New("invalid size")
	ErrNotRunning         = errors.New("worker pool is not running")
)

type State int32

const (
	Created State = iota
	Running
	ShuttingDown
	Stopping
	Stopped
)

type WorkerPool struct {
	jobsBufferSize int
	tasks          chan *domain.Task
	workers        map[uint64]*domain.Worker

	wg sync.WaitGroup
	mu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	state  State

	nextID uint64
}

func NewPool(buffSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &WorkerPool{
		jobsBufferSize: buffSize,
		ctx:            ctx,
		cancel:         cancel,
		state:          Created,
		workers:        make(map[uint64]*domain.Worker),
	}

	return p
}
