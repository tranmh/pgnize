package jobs

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/tranmh/pgnize/internal/store"
)

// Pool runs N workers that drain the recognition queue. For the modest single-box
// deployment the pool lives in-process inside the API binary.
type Pool struct {
	Deps    Deps
	Workers int
	Idle    time.Duration // poll interval when the queue is empty
}

// Run blocks until ctx is cancelled, draining jobs across Workers goroutines.
func (p *Pool) Run(ctx context.Context) {
	if p.Workers < 1 {
		p.Workers = 1
	}
	if p.Idle == 0 {
		p.Idle = 2 * time.Second
	}
	var wg sync.WaitGroup
	for i := 0; i < p.Workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			p.loop(ctx, id)
		}(i)
	}
	wg.Wait()
}

func (p *Pool) loop(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		job, err := p.Deps.Store.ClaimNextJob(ctx)
		if errors.Is(err, store.ErrNotFound) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(p.Idle):
			}
			continue
		}
		if err != nil {
			slog.Error("claim job", "worker", id, "err", err)
			time.Sleep(p.Idle)
			continue
		}
		if err := Process(ctx, p.Deps, job); err != nil {
			slog.Error("process job", "worker", id, "job", job.JobID, "err", err)
		}
	}
}
