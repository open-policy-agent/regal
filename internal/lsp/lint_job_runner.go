package lsp

import (
	"context"
)

type lintJobRunner struct {
	work, stop, done chan struct{}
	lintFunc         func(context.Context)
}

func newLintJobRunner(lintFunc func(context.Context)) *lintJobRunner {
	return &lintJobRunner{
		work:     make(chan struct{}, 1),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		lintFunc: lintFunc,
	}
}

func (l *lintJobRunner) Start(ctx context.Context) {
	l.loop(ctx)
}

func (l *lintJobRunner) Stop(ctx context.Context) error {
	close(l.stop)

	select {
	case <-l.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *lintJobRunner) Trigger() {
	select {
	case l.work <- struct{}{}:
	default:
	}
}

func (l *lintJobRunner) loop(ctx context.Context) {
	defer close(l.done)

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.stop:
			return
		case <-l.work:
			l.lintFunc(ctx)
		}
	}
}
