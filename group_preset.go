package safegroup

import (
	"context"
	"sync"
)

// GroupPreset creates groups with preconfigured options.
//
// It can also run detached fire-and-forget tasks with the same option set.
type GroupPreset struct {
	mutex sync.RWMutex
	opts  []Option
}

// NewGroupPreset returns a pointer preset that applies opts to each group.
func NewGroupPreset(opts ...Option) *GroupPreset {
	copied := make([]Option, len(opts))
	copy(copied, opts)
	return &GroupPreset{opts: copied}
}

// Group creates a new Group using the preset's configured options.
//
// When p is nil, this method falls back to WithContext(ctx) with default
// options.
func (p *GroupPreset) Group(ctx context.Context) (*Group, context.Context) {
	if p == nil {
		return WithContext(ctx)
	}

	p.mutex.RLock()
	opts := make([]Option, len(p.opts))
	copy(opts, p.opts)
	p.mutex.RUnlock()

	return WithContext(ctx, opts...)
}

// Go starts one detached task using the preset's configured options and the
// provided parent context.
//
// The task does not receive context.Context. Use GoContext when task code
// needs the derived context directly.
//
// This method returns immediately.
// When p is nil, this method uses default options.
func (p *GroupPreset) Go(ctx context.Context, task func() error) {
	p.GoLabel(ctx, "", task)
}

// GoContext starts one detached task using the preset's configured options and
// the provided parent context.
//
// The task receives the group's derived context.
//
// This method returns immediately.
// When p is nil, this method uses default options.
func (p *GroupPreset) GoContext(ctx context.Context, task func(context.Context) error) {
	p.GoLabelContext(ctx, "", task)
}

// GoLabel starts one detached labeled task using the preset's configured
// options and the provided parent context.
//
// The task does not receive context.Context. Use GoLabelContext when task code
// needs the derived context directly.
//
// This method returns immediately.
// When p is nil, this method uses default options.
func (p *GroupPreset) GoLabel(ctx context.Context, label string, task func() error) {
	if task == nil {
		panic("safegroup: nil task")
	}
	p.GoLabelContext(ctx, label, func(context.Context) error {
		return task()
	})
}

// GoLabelContext starts one detached labeled task using the preset's
// configured options and the provided parent context.
//
// The task receives the group's derived context.
// Because the provided parent context is used as the group root, cancellation
// of that parent can stop detached work earlier than expected.
// The group context is canceled after the detached task exits.
//
// This method returns immediately.
// When p is nil, this method uses default options.
func (p *GroupPreset) GoLabelContext(ctx context.Context, label string, task func(context.Context) error) {
	if task == nil {
		panic("safegroup: nil task")
	}

	group, _ := p.Group(ctx)
	group.setOnTaskExit(group.cancel)
	group.GoLabel(label, task)
}

// WithOptions appends options to the preset and returns itself.
//
// Later options take precedence over earlier options for the same setting.
// When p is nil, this method creates and returns a new preset.
func (p *GroupPreset) WithOptions(opts ...Option) *GroupPreset {
	if p == nil {
		return NewGroupPreset(opts...)
	}

	p.mutex.Lock()
	for _, opt := range opts {
		if opt != nil {
			p.opts = append(p.opts, opt)
		}
	}
	p.mutex.Unlock()
	return p
}
