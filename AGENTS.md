# Repository Guidelines

## Project Structure & Module Organization

- Go CLI: `main.go` contains the `robotdiff` entrypoint and core diff/report generation.
- Embedded report UI: `template/` holds `report.html`, `styles.css`, and `app.js` which are embedded via Go `//go:embed` to produce a single self-contained HTML report.
- Sample inputs: `test1.xml`, `test2.xml` are example Robot Framework output files.
- Generated output: default report file is `robotdiff.html`.

Key commands:

- `go build -o robotdiff` — builds the local binary.
- `./robotdiff test1.xml test2.xml` — generates `robotdiff.html` from inputs.
- `go install ./@latest` — installs the CLI into `$(go env GOPATH)/bin` (Go 1.17+).

## Coding Style & Naming Conventions

- Go: use `gofmt` formatting (tabs for indentation) and idiomatic naming (exported `PascalCase`, local `camelCase`). Prefer small, focused helpers over deeply nested logic.
- Front-end assets: keep `template/app.js` readable and function-oriented; avoid adding dependencies unless there’s a clear payoff since the report is intended to be offline-ready.

## Testing Guidelines

- JavaScript harness: open `template/test.html` in a browser; it runs assertions from `template/app.test.js`.
- Go tests: none yet. New logic should ideally be covered with `go test ./...` once tests are introduced (e.g., status calculation, row-building).

Naming:

- Go tests: `*_test.go` (e.g., `row_status_test.go`).
- JS tests: keep related assertions in `template/app.test.js`.

## Commit & Pull Request Guidelines

- Commits are short, imperative summaries (examples in history: “Add history”, “Update readme”). Keep subject lines concise and scoped.
- PRs should include: a clear description, before/after screenshots of the HTML report for UI changes, and a quick local verification note (e.g., command + input files used).

## Security & Configuration Tips

- Treat Robot XML inputs as untrusted: avoid introducing template injection paths and keep output HTML self-contained.
- History mode writes to `--history-file`; don’t commit generated history JSON or `robotdiff.html` unless explicitly required.
