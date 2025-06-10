package pool

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

type mockJob struct {
	id          int
	duration    time.Duration
	err         error
	shouldPanic bool
	payload     any
}

func (j *mockJob) Execute(ctx context.Context) (any, error) {
	if j.shouldPanic {
		panic("test panic")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(j.duration):
		return j.payload, j.err
	}
}

type mockBlockingJob struct {
	unblock chan struct{}
	started chan struct{}
}

func newMockBlockingJob() *mockBlockingJob {
	return &mockBlockingJob{
		unblock: make(chan struct{}),
		started: make(chan struct{}),
	}
}

func (j *mockBlockingJob) Execute(ctx context.Context) (any, error) {
	close(j.started)
	select {
	case <-j.unblock:
		return "unblocked", nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestNewPool(t *testing.T) {
	t.Parallel()
	p := NewPool(10)
	require.NotNil(t, p)
	assert.Equal(t, Created, p.atomicLoadState())
	assert.Equal(t, 10, p.jobsBufferSize)
	assert.NotNil(t, p.ctx)
	assert.NotNil(t, p.cancel)
	assert.Empty(t, p.workers)
}

func TestWorkerPool_Start(t *testing.T) {
	t.Parallel()

	t.Run("SuccessfulStart", func(t *testing.T) {
		t.Parallel()
		p := NewPool(10)
		err := p.Start(5)
		require.NoError(t, err)
		defer p.Shutdown()

		assert.Equal(t, Running, p.atomicLoadState())
		assert.NotNil(t, p.tasks)
		require.Eventually(t, func() bool {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return len(p.workers) == 5
		}, 100*time.Millisecond, 10*time.Millisecond)
	})

	t.Run("StartWithNegativePoolSize", func(t *testing.T) {
		t.Parallel()
		p := NewPool(10)
		err := p.Start(-1)
		require.NoError(t, err)
		defer p.Shutdown()

		assert.Equal(t, Running, p.atomicLoadState())
		p.mu.RLock()
		assert.Len(t, p.workers, 0)
		p.mu.RUnlock()
	})

	t.Run("StartAlreadyRunningPool", func(t *testing.T) {
		t.Parallel()
		p := NewPool(10)
		err := p.Start(1)
		require.NoError(t, err)
		defer p.Shutdown()

		err = p.Start(1)
		assert.ErrorIs(t, err, ErrPoolAlreadyRunning)
	})
}

func TestWorkerPool_Submit(t *testing.T) {
	t.Parallel()

	t.Run("SubmitSuccessfulJob", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		require.NoError(t, p.Start(1))
		defer p.Shutdown()

		job := &mockJob{payload: "success"}
		future, err := p.Submit(context.Background(), job)
		require.NoError(t, err)
		require.NotNil(t, future)

		res, err := future.Wait(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "success", res)
	})

	t.Run("SubmitFailingJob", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		require.NoError(t, p.Start(1))
		defer p.Shutdown()

		expectedErr := errors.New("job failed")
		job := &mockJob{err: expectedErr}
		future, err := p.Submit(context.Background(), job)
		require.NoError(t, err)
		require.NotNil(t, future)

		res, err := future.Wait(context.Background())
		assert.ErrorIs(t, err, expectedErr)
		assert.Nil(t, res)
	})

	t.Run("SubmitJobThatPanics", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		require.NoError(t, p.Start(1))
		defer p.Shutdown()

		job := &mockJob{shouldPanic: true}
		future, err := p.Submit(context.Background(), job)
		require.NoError(t, err)
		require.NotNil(t, future)

		res, err := future.Wait(context.Background())
		assert.ErrorIs(t, err, ErrJobPanic)
		assert.Nil(t, res)
	})

	t.Run("SubmitWithCancelledContext", func(t *testing.T) {
		t.Parallel()
		p := NewPool(0)
		require.NoError(t, p.Start(0))
		defer p.Shutdown()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		job := &mockJob{}
		future, err := p.Submit(ctx, job)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, future)
	})

	t.Run("SubmitToStoppedPool", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		require.NoError(t, p.Start(1))
		p.Stop()

		job := &mockJob{}
		future, err := p.Submit(context.Background(), job)
		assert.ErrorIs(t, err, ErrPoolStopped)
		assert.Nil(t, future)
	})
}

func TestWorkerPool_ShutdownAndStop(t *testing.T) {
	t.Parallel()

	t.Run("ShutdownWaitsForJobsToComplete", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		require.NoError(t, p.Start(1))

		job := &mockJob{duration: 50 * time.Millisecond, payload: "done"}
		future, err := p.Submit(context.Background(), job)
		require.NoError(t, err)

		p.Shutdown()

		assert.Equal(t, Stopped, p.atomicLoadState())
		res, err := future.Wait(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "done", res)
	})

}

func TestWorkerPool_Resize(t *testing.T) {
	t.Parallel()

	t.Run("ResizeNotRunningPool", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		err := p.Resize(5)
		assert.ErrorIs(t, err, ErrNotRunning)
	})

	t.Run("ResizeWithInvalidSize", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		require.NoError(t, p.Start(2))
		defer p.Shutdown()

		err := p.Resize(-1)
		assert.ErrorIs(t, err, ErrInvalidSize)
	})

	t.Run("IncreaseAndDecreaseWorkers", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		require.NoError(t, p.Start(2))
		defer p.Shutdown()

		err := p.Resize(5)
		assert.NoError(t, err)
		assert.Eventually(t, func() bool {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return len(p.workers) == 5
		}, 100*time.Millisecond, 10*time.Millisecond, "should have 5 workers")

		err = p.Resize(3)
		assert.NoError(t, err)
		assert.Eventually(t, func() bool {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return len(p.workers) == 3
		}, 100*time.Millisecond, 10*time.Millisecond, "should have 3 workers")
	})
}

func TestWorkerPool_ResizeBuffer(t *testing.T) {
	t.Parallel()

	t.Run("ResizeBufferOnNonRunningPool", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		err := p.ResizeBuffer(5)
		assert.ErrorIs(t, err, ErrNotRunning)
	})

	t.Run("ResizeBufferWithInvalidSize", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		require.NoError(t, p.Start(1))
		defer p.Shutdown()

		err := p.ResizeBuffer(-1)
		assert.ErrorIs(t, err, ErrInvalidSize)
	})

	t.Run("TaskMigrationAndDropOnBufferResize", func(t *testing.T) {
		t.Parallel()
		p := NewPool(2)
		require.NoError(t, p.Start(1))
		defer p.Shutdown()

		bJob := newMockBlockingJob()
		_, err := p.Submit(context.Background(), bJob)
		require.NoError(t, err)
		<-bJob.started

		f1, err := p.Submit(context.Background(), &mockJob{payload: 1})
		require.NoError(t, err)
		f2, err := p.Submit(context.Background(), &mockJob{payload: 2})
		require.NoError(t, err)

		err = p.ResizeBuffer(0)
		require.NoError(t, err)

		close(bJob.unblock)

		var wg sync.WaitGroup
		wg.Add(2)
		var errs [2]error

		go func() {
			defer wg.Done()
			_, errs[0] = f1.Wait(context.Background())
		}()
		go func() {
			defer wg.Done()
			_, errs[1] = f2.Wait(context.Background())
		}()
		wg.Wait()

		assert.ErrorIs(t, errs[0], ErrPoolStopped, "first queued job should be dropped")
	})
}
