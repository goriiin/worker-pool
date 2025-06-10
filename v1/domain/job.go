package domain

import (
	"context"
)

type Job interface {
	Execute(ctx context.Context) (any, error)
}

type Task struct {
	Job    Job
	Ctx    context.Context
	Future *Future
}

type JobFunc func(ctx context.Context) (any, error)

func (f JobFunc) Execute(ctx context.Context) (any, error) {
	return f(ctx)
}
