package safegroup

// DefaultPreset is used by package-level Go and GoLabel helpers.
//
// Configure it with WithXXX methods when you need non-default behavior.
var DefaultPreset = NewGroupPreset()

// Go starts one detached task using DefaultPreset.
func Go(task func() error) {
	DefaultPreset.Go(task)
}

// GoLabel starts one detached labeled task using DefaultPreset.
func GoLabel(label string, task func() error) {
	DefaultPreset.GoLabel(label, task)
}
