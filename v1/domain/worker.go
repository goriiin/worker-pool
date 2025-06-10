package domain

import "context"

type Worker struct {
	ID     uint64
	Cancel context.CancelFunc
}
