package pool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goriiin/worker-pool/v1/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testJob struct {
	id          int
	duration    time.Duration
	shouldPanic bool
	shouldErr   bool
}

func (j *testJob) Execute(ctx context.Context) (any, error) {
	if j.shouldPanic {
		panic("test panic")
	}

	select {
	case <-time.After(j.duration):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if j.shouldErr {
		return nil, fmt.Errorf("job %d failed as requested", j.id)
	}

	return fmt.Sprintf("job %d success", j.id), nil
}

func TestPool_Lifecycle(t *testing.T) {
	p := NewPool(10)
	err := p.Start(2)
	require.NoError(t, err)

	future, err := p.Submit(context.Background(), &testJob{id: 1})
	require.NoError(t, err)

	res, err := future.Wait(context.Background())
	require.NoError(t, err, "waiting for the result should not cause an error")
	assert.Equal(t, "job 1 success", res)

	p.Shutdown()

	_, err = p.Submit(context.Background(), &testJob{id: 1})
	assert.ErrorIs(t, err, ErrPoolStopped)
}

func TestPool_StartErrors(t *testing.T) {
	t.Parallel()
	t.Run("Start already running pool", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)

		err := p.Start(1)
		require.NoError(t, err)

		defer p.Shutdown()

		err = p.Start(1)
		assert.ErrorIs(t, err, ErrPoolAlreadyRunning)
	})

	t.Run("Start with negative size", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)

		err := p.Start(-1)
		require.NoError(t, err)
		defer p.Shutdown()

		assert.Len(t, p.workers, 0)
	})
}

func TestPool_Resize(t *testing.T) {
	p := NewPool(10)
	p.Start(2)
	defer p.Shutdown()

	require.Len(t, p.workers, 2)

	require.NoError(t, p.Resize(5))
	require.Eventually(t, func() bool {
		p.mu.RLock()
		defer p.mu.RUnlock()
		return len(p.workers) == 5
	}, time.Second, 50*time.Millisecond)

	require.NoError(t, p.Resize(3))

	require.Eventually(t, func() bool {
		p.mu.RLock()
		defer p.mu.RUnlock()
		return len(p.workers) == 3
	}, time.Second, 50*time.Millisecond)

	err := p.Resize(-1)
	assert.ErrorIs(t, err, ErrInvalidSize)

	p.Shutdown()
	err = p.Resize(10)
	assert.ErrorIs(t, err, ErrPoolStopped)
}

func TestPool_ResizeBuffer(t *testing.T) {
	p := NewPool(1)
	p.Start(1)
	defer p.Shutdown()

	jobStarted := make(chan struct{})
	jobCanFinish := make(chan struct{})

	blockingJob := domain.JobFunc(func(ctx context.Context) (interface{}, error) {
		close(jobStarted)
		<-jobCanFinish

		return "done", nil
	})

	_, err := p.Submit(context.Background(), blockingJob)
	require.NoError(t, err)
	<-jobStarted

	_, err = p.Submit(context.Background(), &testJob{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = p.Submit(ctx, &testJob{})
	require.ErrorIs(t, err, context.DeadlineExceeded)

	require.NoError(t, p.ResizeBuffer(2))
	assert.Equal(t, 2, p.jobsBufferSize)

	_, err = p.Submit(context.Background(), &testJob{})
	require.NoError(t, err)

	close(jobCanFinish)
}

func TestParallel_JobOutcomes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		job              domain.Job
		expectedErrorMsg string
		expectedErrorIs  error
	}{
		{
			name:             "Job returns an error",
			job:              &testJob{shouldErr: true, id: 1},
			expectedErrorMsg: "job 1 failed as requested",
		},
		{
			name:            "Job causes a panic",
			job:             &testJob{shouldPanic: true},
			expectedErrorIs: ErrJobPanic,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewPool(1)
			p.Start(1)
			defer p.Shutdown()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			future, err := p.Submit(ctx, tc.job)
			require.NoError(t, err)

			if tc.name == "Job context is cancelled before execution" {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}

			_, err = future.Wait(context.Background())
			require.Error(t, err)

			if tc.expectedErrorMsg != "" {
				assert.EqualError(t, err, tc.expectedErrorMsg)
			}
			if tc.expectedErrorIs != nil {
				assert.ErrorIs(t, err, tc.expectedErrorIs)
			}
		})
	}
}

func TestParallel_ShutdownVsStop(t *testing.T) {
	t.Parallel()

	t.Run("Shutdown waits for jobs", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		p.Start(1)

		var counter int32
		job := domain.JobFunc(func(ctx context.Context) (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(&counter, 1)
			return nil, nil
		})
		_, err := p.Submit(context.Background(), job)
		require.NoError(t, err)

		p.Shutdown()

		assert.Equal(t, int32(1), atomic.LoadInt32(&counter), "Counter should be 1 after graceful shutdown")
		assert.Equal(t, Stopped, p.state)
	})
}

func TestParallel_Idempotency(t *testing.T) {
	t.Parallel()

	t.Run("Concurrent Shutdown calls", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		p.Start(1)
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.Shutdown()
			}()
		}
		wg.Wait()
		assert.Equal(t, Stopped, p.state)
	})

	t.Run("Concurrent Stop calls", func(t *testing.T) {
		t.Parallel()
		p := NewPool(1)
		p.Start(1)
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.Stop()
			}()
		}
		wg.Wait()
		assert.Equal(t, Stopped, p.state)
	})
}
