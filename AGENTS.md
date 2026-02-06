# AGENTS.md

## Scope
This file applies to the entire repository.

## Project Summary
- Module: `github.com/yacchi/safegroup`
- Purpose: panic-safe, join-first goroutine group for Go 1.20+
- Key behavior:
  - Recovers panics and converts them to `*PanicError`
  - Returns all failures from `Wait()` via `errors.Join`

## Development Environment
- Use `mise` for Go toolchains.
- If `mise` warns about untrusted config, run:
  - `mise trust`
- Default toolchain is managed in `mise.toml`.

## Task Runner
- Primary entrypoint is `Makefile`.
- Common commands:
  - `make test` (single Go version)
  - `make test-matrix` (Go 1.20, 1.21, 1.22, 1.23, 1.24)
  - `make fmt`

## Test Policy
- Run `make test` after code changes.
- For compatibility-sensitive changes, run `make test-matrix`.
- Do not change unrelated tests to make a change pass.

## Code Style
- Keep changes minimal and focused.
- Prefer clear APIs over compatibility shims.
- Preserve current semantics unless explicitly requested:
  - `Wait()` remains join-first
  - panic handling remains structured through `PanicError`
- Use `gofmt` for all Go source changes.

## Documentation Policy
- All exported types/functions/methods must have GoDoc comments.
- Keep README concise and defer canonical API details to `pkg.go.dev`.
- Reflect behavior changes in both GoDoc and README when needed.

## Commit Messages
- All commit messages must be written in English.
- Use Conventional Commit format (e.g., `feat:`, `fix:`, `docs:`, `refactor:`).

## Non-Goals
- This library is not intended to be `errgroup` return-compatible.
- `runtime.Goexit` is not supported.
