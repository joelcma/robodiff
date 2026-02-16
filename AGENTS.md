# Repository Guidelines

## Project Structure & Module Organization

- Go CLI: `main.go` contains the `robodiff` entrypoint (server mode).
- Go backend: `backend/server/` is the HTTP API + static file serving for the React build.
- Robot parsing/diff: `backend/diff/` parses Robot output.xml and computes comparisons.
- React frontend: `web/` contains the UI (Vite dev server; production build to `web/dist/`).

Key commands:

- `go build -o robodiff` — builds the local binary.
- `./robodiff --dir /path/to/results` — starts the server.
- `cd web && npm install && npm run dev` — runs the UI dev server.
- `cd web && npm run build` — builds UI assets to `web/dist/`.
- `go install ./@latest` — installs the CLI into `$(go env GOPATH)/bin` (Go 1.17+).

## Coding Style & Naming Conventions

- Go: use `gofmt` formatting (tabs for indentation) and idiomatic naming (exported `PascalCase`, local `camelCase`). Prefer small, focused helpers over deeply nested logic.
- Frontend: keep `web/src` components small and function-oriented; avoid adding dependencies unless there’s a clear payoff.

## Testing Guidelines

- Frontend: `cd web && npm run lint` and `cd web && npm run build`.
- Go tests: none yet. New logic should ideally be covered with `go test ./...` once tests are introduced.

Naming:

- Go tests: `*_test.go` (e.g., `row_status_test.go`).
- JS tests: none currently.

## Commit & Pull Request Guidelines

- Commits are short, imperative summaries (examples in history: “Add history”, “Update readme”). Keep subject lines concise and scoped.
- PRs should include: a clear description, before/after screenshots of the HTML report for UI changes, and a quick local verification note (e.g., command + input files used).

## Security & Configuration Tips

- Treat Robot XML inputs as untrusted: avoid introducing template injection paths and keep output HTML self-contained.
