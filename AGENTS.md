# Project instructions for AI coding agents

## Commands

### After all changes are made, run

- `mise run test`
- `mise run check`
- `go fix [changed_files]...`

## Tech stack

- go (look at go.mod for version)
- gRPC
- protobuf
- wireguard
- sqlite

## Code style

- Write obvious code instead of clever code.
- Favor self-explanatory code over code comments.
- If you really have to add a comment, make sure it's short and concise.
- Wrap errors with context: `fmt.Errorf("short description: %w", err)`.
- Use `testify/require` and `testify/assert` for tests. Prefer table-driven tests.
- Platform-specific code goes in files with build-tag suffixes (`_darwin.go`, `_linux.go`, `_windows.go`).

## Git

- Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for all commit messages.
- Never commit unless explicitly told to.
- Never push unless explicitly told to.
