package main

import (
	"context"
	"fmt"
	"github.com/goriiin/worker-pool/v1/config"
	"github.com/goriiin/worker-pool/v1/domain"
	"github.com/goriiin/worker-pool/v1/pool"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type myJob struct {
	id  int
	str string
}

const path = "./v1/config/config.yml"

func (m myJob) Execute(ctx context.Context) (any, error) {
	fmt.Println(m.id, m.str)

	return fmt.Sprintf("my str: %s", m.str), nil
}

func main() {
	if err := config.Load(path, nil); err != nil {
		log.Fatalf("config file not found err: %v, path^ %s", err, path)
	}

	w := pool.NewPool(config.Conf.Pool.Buffer)

	onChanged := func(c config.Config) {
		if err := w.Resize(c.Pool.Size); err != nil {
			log.Printf("resize workers err: %v", err)
		}

		if err := w.ResizeBuffer(c.Pool.Buffer); err != nil {
			log.Printf("resize buffer err: %v", err)
		}

		log.Printf("resize workers done buffer: %d workers: %d", c.Pool.Size, c.Pool.Buffer)
	}

	if err := config.Load(path, onChanged); err != nil {
		log.Fatalf("config file not found err: %v, path^ %s", err, path)
	}

	err := w.Start(config.Conf.Pool.Size)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < 1000; i++ {
		fmt.Println(i)

		time.Sleep(time.Millisecond * 500)

		job := &myJob{
			id:  i,
			str: fmt.Sprintf("test: %d", i*i),
		}

		fut, err := w.Submit(ctx, job)
		if err != nil {
			log.Printf("submit err: %v", err)

			continue
		}

		go func(f *domain.Future, id int) {
			res, err := f.Wait(ctx)
			if err != nil {
				log.Printf("id: %d, wait err: %v", id, err)

				return
			}

			log.Printf("id: %d, wait result: %v", id, res)
		}(fut, i)
	}

	log.Println("press ctrl+c for exit")
	q := make(chan os.Signal, 1)
	signal.Notify(q, syscall.SIGINT, syscall.SIGTERM)

	<-q

	w.Shutdown()
	log.Println("shutting down success")
}
