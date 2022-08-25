package rotator

import (
	"context"
	"sync"
	"time"

	"github.com/polarismesh/polaris-sidecar/log"
)

type Rotator struct {
	once       sync.Once
	period     time.Duration
	retryDelay time.Duration
}

func New(period time.Duration, retryDelay time.Duration) *Rotator {
	return &Rotator{
		period:     period,
		retryDelay: retryDelay,
	}
}

func (r *Rotator) init() {
	if r.period == 0 {
		r.period = time.Minute * 30
	}

	if r.retryDelay == 0 {
		r.retryDelay = time.Second
	}
}

func (r *Rotator) execute(ctx context.Context, f func(ctx context.Context) error) {
	for {
		log.Infof("will execute by rotator")
		err := f(ctx)
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return
		}
		log.Errorf("action executed failed: %s", err.Error())
		time.Sleep(r.retryDelay)
	}
}

func (r *Rotator) Run(ctx context.Context, f func(ctx context.Context) error) error {
	r.once.Do(r.init)
	r.execute(ctx, f)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.period):
		}

		r.execute(ctx, f)

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}
