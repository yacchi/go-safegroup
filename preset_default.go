package safegroup

import "context"

// DefaultPreset is used by package-level Go and GoLabel helpers.
//
// Configure it with WithXXX methods when you need non-default behavior.
// Replacing this global variable while tasks are running can race; configure
// it during initialization/startup before concurrent use.
var DefaultPreset = NewGroupPreset()

// DefaultGroup creates a new Group using DefaultPreset and the provided parent
// context.
func DefaultGroup(ctx context.Context) (*Group, context.Context) {
	return DefaultPreset.Group(ctx)
}

// Go starts one detached task using DefaultPreset and the provided parent
// context.
//
// The task does not receive context.Context. Use GoContext when task code
// needs the derived context directly.
func Go(ctx context.Context, task func() error) {
	DefaultPreset.Go(ctx, task)
}

// GoContext starts one detached task using DefaultPreset and the provided
// parent context.
//
// The task receives the derived context.
func GoContext(ctx context.Context, task func(context.Context) error) {
	DefaultPreset.GoContext(ctx, task)
}

// GoLabel starts one detached labeled task using DefaultPreset and the
// provided parent context.
//
// The task does not receive context.Context. Use GoLabelContext when task code
// needs the derived context directly.
func GoLabel(ctx context.Context, label string, task func() error) {
	DefaultPreset.GoLabel(ctx, label, task)
}

// GoLabelContext starts one detached labeled task using DefaultPreset and the
// provided parent context.
//
// The task receives the derived context.
func GoLabelContext(ctx context.Context, label string, task func(context.Context) error) {
	DefaultPreset.GoLabelContext(ctx, label, task)
}
