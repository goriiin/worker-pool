package domain

import "context"

type FutureResult struct {
	Res any
	Err error
}

type Future struct {
	Result chan FutureResult
}

func (f *Future) Wait(ctx context.Context) (any, error) {
	select {
	case result := <-f.Result:
		return result.Res, result.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
