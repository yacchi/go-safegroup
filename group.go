package safegroup

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type token struct{}

// Group runs tasks with a shared context and collects all returned errors and
// recovered panics.
type Group struct {
	ctx    context.Context
	cancel context.CancelFunc
	cfg    config

	sem chan token

	waitGroup sync.WaitGroup
	closed    atomic.Bool

	mutex  sync.Mutex
	errors []error
	panics []*PanicError
}

// WithContext creates a new Group and derived context.
//
// The returned context is canceled when:
//   - the parent context is canceled,
//   - a task returns an error and CancelOnError(true),
//   - a task panics and CancelOnPanic(true),
//   - Wait returns.
func WithContext(ctx context.Context, opts ...Option) (*Group, context.Context) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	groupContext, cancel := context.WithCancel(ctx)
	group := &Group{
		ctx:    groupContext,
		cancel: cancel,
		cfg:    cfg,
	}
	return group, groupContext
}

// Go starts a task without a label.
//
// This method panics after Wait has returned.
func (g *Group) Go(task func(context.Context) error) {
	g.GoLabel("", task)
}

// GoLabel starts a labeled task.
//
// The label is copied into PanicError when the task panics.
// When a limit is set with SetLimit, this method blocks until a slot is
// available or the group context is canceled. If the context is canceled
// while waiting, the task is not started.
//
// This method panics after Wait has returned.
func (g *Group) GoLabel(label string, task func(context.Context) error) {
	if task == nil {
		panic("safegroup: nil task")
	}
	if g.closed.Load() {
		panic("safegroup: Go after Wait")
	}
	if g.sem != nil {
		select {
		case g.sem <- token{}:
			if g.closed.Load() {
				<-g.sem
				panic("safegroup: Go after Wait")
			}
			select {
			case <-g.ctx.Done():
				<-g.sem
				return
			default:
			}
		case <-g.ctx.Done():
			return
		}
	}
	g.waitGroup.Add(1)
	go g.runTask(label, task)
}

// TryGo starts a task without a label if the concurrency limit allows it.
//
// It returns false when SetLimit is configured and no slot is currently
// available.
// It also returns false after Wait has returned.
func (g *Group) TryGo(task func(context.Context) error) bool {
	return g.TryGoLabel("", task)
}

// TryGoLabel starts a labeled task if the concurrency limit allows it.
//
// It returns false when SetLimit is configured and no slot is currently
// available.
// It also returns false after Wait has returned.
func (g *Group) TryGoLabel(label string, task func(context.Context) error) bool {
	if task == nil {
		panic("safegroup: nil task")
	}
	if g.closed.Load() {
		return false
	}
	if g.sem != nil {
		select {
		case g.sem <- token{}:
			if g.closed.Load() {
				<-g.sem
				return false
			}
		default:
			return false
		}
	}
	g.waitGroup.Add(1)
	go g.runTask(label, task)
	return true
}

// SetLimit sets the maximum number of active tasks.
//
// A zero limit removes the limit.
// Call SetLimit only before starting tasks.
func (g *Group) SetLimit(limit int) {
	if limit < 0 {
		panic("safegroup: negative limit")
	}
	if g.sem != nil && len(g.sem) != 0 {
		panic("safegroup: SetLimit while tasks are active")
	}

	if limit == 0 {
		g.sem = nil
		return
	}
	g.sem = make(chan token, limit)
}

// Wait blocks until all started tasks finish and returns all collected failures
// as errors.Join.
//
// It returns nil when no task failed and no task panicked.
// Wait may be called multiple times and returns a consistent snapshot of
// collected failures.
// Wait cancels the group context before returning. Tasks started after a Wait
// call are rejected.
func (g *Group) Wait() error {
	g.waitGroup.Wait()
	g.cancel()
	g.closed.Store(true)

	g.mutex.Lock()
	defer g.mutex.Unlock()

	if len(g.errors) == 0 {
		return nil
	}

	joined := make([]error, len(g.errors))
	copy(joined, g.errors)
	return errors.Join(joined...)
}

// Errors returns a snapshot of all collected failures in append order.
//
// Returned errors include regular task errors and PanicError values.
func (g *Group) Errors() []error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	result := make([]error, len(g.errors))
	copy(result, g.errors)
	return result
}

// Panics returns a snapshot of collected panic errors in append order.
func (g *Group) Panics() []*PanicError {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	result := make([]*PanicError, len(g.panics))
	copy(result, g.panics)
	return result
}

func (g *Group) runTask(label string, task func(context.Context) error) {
	defer g.waitGroup.Done()
	if g.sem != nil {
		defer func() {
			<-g.sem
		}()
	}

	if err := g.runTaskBody(label, task); err != nil {
		g.appendError(err)
		if g.cfg.cancelOnError {
			g.cancel()
		}
		g.cfg.onError(err)
	}
}

func (g *Group) runTaskBody(label string, task func(context.Context) error) (taskErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			panicError := &PanicError{
				Label: label,
				Value: recovered,
			}
			if g.cfg.captureStack {
				panicError.Stack = captureStack(2)
			}
			g.appendPanic(panicError)
			if g.cfg.cancelOnPanic {
				g.cancel()
			}
			g.cfg.onPanic(panicError)
		}
	}()

	return task(g.ctx)
}

func (g *Group) appendError(err error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.errors = append(g.errors, err)
}

func (g *Group) appendPanic(panicError *PanicError) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.errors = append(g.errors, panicError)
	g.panics = append(g.panics, panicError)
}
