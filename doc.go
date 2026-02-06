// Package safegroup provides a panic-safe, join-first goroutine group.
//
// Unlike errgroup-style "first error wins" behavior, safegroup collects all
// task failures (including recovered panics) and returns them via errors.Join.
package safegroup
