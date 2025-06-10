package pool

import "github.com/goriiin/worker-pool/v1/domain"

func (wp *WorkerPool) execute(t *domain.Task) {
	defer func() {
		if r := recover(); r != nil {
			t.Future.Result <- domain.FutureResult{
				Err: ErrJobPanic,
				Res: nil,
			}

			close(t.Future.Result)
		}
	}()

	if t.Ctx.Err() != nil {
		t.Future.Result <- domain.FutureResult{Err: t.Ctx.Err()}

		return
	}

	res, err := t.Job.Execute(t.Ctx)
	select {
	case t.Future.Result <- domain.FutureResult{
		Res: res,
		Err: err}:
		return

	case <-t.Ctx.Done():
		return
	}

}
