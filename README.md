# Robot Diff — Go edition

Fast, zero-dependency, native Go implementation of the Robot Framework output diff tool. It compares two or more Robot Framework output XML files and generates a single, self-contained HTML diff report.

## Highlights

- Compare 2+ Robot Framework output XML files and produce an offline-ready HTML report.
- Single compiled binary; templates and data are embedded so the generated HTML is self-contained.
- Optional history support — visualise trends and save selected runs (localStorage and optional JSON file on disk).

## Quick start (try it)

Build:

```bash
go build -o robotdiff
```

Install (Go 1.17+):

```bash
go install ./@latest
# binary will be available in $GOPATH/bin or $(go env GOPATH)/bin
```

Generate a report using the included example files:

```bash
./robotdiff test1.xml test2.xml
# default output: robotdiff.html
open robotdiff.html
```

## Usage / Flags

```
Usage: robotdiff [options] input_files

Flags:
 -r, --report <file>       HTML report file (default: robotdiff.html)
 -n, --name <name>         Name for test runs; repeat for each input file
 -t, --title <title>       Report title (default: "Test Run Diff Report")
 --enable-history          Enable history support in the generated report
 --history-file <file>     Path to history JSON file (default: robotdiff_history.json)
 -h, --help                Show help
```

Examples:

```bash
# Compare three outputs
./robotdiff out1.xml out2.xml out3.xml

# Custom names for runs
./robotdiff --name 'CI-main' --name 'release-1' out1.xml out2.xml

# Enable persistent history stored on disk
./robotdiff --enable-history --history-file my_history.json out1.xml out2.xml
```

Note: The CLI parser currently includes a `--view-history` flag but that flag is not implemented in the code — consider it a TODO to either implement or remove.

## Template / embedded assets

The `template/` directory contains the HTML, CSS and JavaScript used to render the report. These files are embedded into the final HTML using Go's `//go:embed` so the generated report is fully self-contained.

Files of interest:

- template/report.html — HTML template with placeholders
- template/styles.css — CSS used in reports
- template/app.js — JS used in the report and history UI
- template/app.test.js — a small JS test harness (open `template/test.html` to run)

## Testing

- JavaScript tests: open `template/test.html` in a browser to run the included `app.test.js` harness.
- Go tests: there are currently no unit tests for the Go logic. It's recommended to add tests for the status-calculation and `Rows()` logic (e.g., `row_status_test.go`).

## CI / recommended automation

- Add a simple GitHub Actions workflow to run `go test` / `go vet` / `golangci-lint` and build artifacts for macOS/Linux/Windows.
- Optionally add release artifacts (prebuilt binaries) for easy installation.

## Contributing

Contributions welcome. Open issues for bugs/feature requests. For code changes, please open a pull request with tests where applicable.

## Compatibility / build

- Requires Go 1.16+ for `//go:embed`. This repo is tested against Go 1.21 (see `go.mod`).

## License

Apache-2.0

This repository is a derivative of the Robot Framework Robot Diff tool (Nokia) and retains attribution to the original authors.
