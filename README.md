# Robodiff — Go + React

Modern web application for viewing and comparing Robot Framework test results. Built with a Go backend and React frontend, it watches a directory for Robot Framework XML files and provides real-time interactive analysis.

## Disclaimer

I've heavily utilized AI coding agents for this project and I have not been overly concerned with perfect code quality or architecture. The focus has been on shipping a functional tool quickly. Please excuse any rough edges in the implementation.

## Highlights

- **Real-time monitoring**: Continuously scans a directory (and subdirectories) for Robot Framework XML files
- **Interactive UI**: React-based web interface with keyboard shortcuts for power users
- **Single run viewer**: Inspect individual test runs with detailed keyword/step information
- **Diff comparison**: Compare multiple runs side-by-side to spot regressions
- **Test details**: Click any test to see execution steps, arguments, messages, and timing
- **API helpers**: Copy curl commands and re-send HTTP requests from API keywords
- **Run cleanup**: Delete selected runs directly from the UI
- **Smart navigation**: Sidebar with suite overview, sticky headers, collapsible sections
- **Go stdlib backend**: No external Go dependencies; UI is served from `web/dist`

## Quick start

Build:

```bash
go build -o robodiff
```

Install (Go 1.17+):

```bash
go install ./@latest
# binary will be available in $GOPATH/bin or $(go env GOPATH)/bin
```

Start the server:

```bash
# Watch a directory for Robot Framework XML files
./robodiff /path/to/robot/results

# Or use current directory
./robodiff .

# Custom port
./robodiff --addr :3000 /path/to/results
```

Then open http://localhost:8080 in your browser (or whatever address you set).

## Usage / Flags

```
Usage: robodiff [options] [<directory>]

Starts HTTP server and watches directory for Robot Framework XML files.

  --addr <address>       HTTP server address (default: :8080)
  --dir <path>           Directory to watch (alternative to positional argument)
  --scan-interval <dur>  Directory scan interval (default: 2s)
  -h, --help             Show help
```

Examples:

```bash
# Start server watching /tmp/robot_runs
./robodiff /tmp/robot_runs

# Custom port
./robodiff --addr :3000 /tmp/robot_runs


```

## Features

### Run Management

- **Auto-discovery**: Scans the directory (up to depth 3, including symlinked dirs) for Robot XML files
- **Search & filter**: Find runs by name or path
- **Sort**: By modification time, size, or test counts
- **Multi-select**: Select specific runs to compare
- **Quick actions**: Select all, select failed, clear selection
- **Delete runs**: Remove selected runs from disk

### Single Run View

- View all tests in a single run
- Filter by status (All/Passed/Failed)
- Suite sidebar with pass/fail counts
- Click any test to see detailed execution steps
- Collapsible run list for maximum screen space

### Test Details Panel

- Full keyword hierarchy with nested steps
- Keyword arguments (e.g., comment text)
- Log messages with timestamps and levels (INFO/WARN/FAIL)
- Execution timing for each keyword
- Right-side panel keeps test list in context

### Diff Comparison

- Compare 2+ runs side-by-side
- Color-coded status changes (Pass→Fail, Fail→Pass, Missing)
- Filter by differences or failures only
- Suite-by-suite comparison with collapsible sections

### Keyboard Shortcuts

- `?` — Show keyboard shortcuts help
- `R` — Refresh runs
- `Ctrl+A` — Select all runs
- `C` — Clear selection
- `F` — Select only failed runs
- `Ctrl+D` — Compare/view selected runs
- `Esc` — Close current view

## Architecture

### Backend (Go)

- **HTTP server**: REST API for run data and test details
- **Folder scanner**: Watches directory every 2 seconds for changes
- **XML parser**: Parses Robot Framework XML on demand
- **Endpoints**:
  - `GET /api/health` — Health check
  - `GET /api/config` — Server configuration
  - `GET /api/runs` — List available runs
  - `POST /api/delete-runs` — Delete runs by ID
  - `POST /api/run` — Get single run details
  - `POST /api/test-details` — Get test execution details
  - `POST /api/http-try` — Execute an HTTP request captured from logs
  - `POST /api/diff` — Compare multiple runs

### Frontend (React)

- **Modern UI**: Component-based architecture
- **Vite dev server**: Fast development with hot reload
- **Production build**: Static assets served by Go backend
- **Key components**:
  - `Header` — Top bar with controls
  - `RunList` — Run selection table
  - `SingleRunView` — Individual run viewer
  - `DiffView` — Comparison results
  - `Sidebar` — Suite navigation
  - `TestDetailsPanel` — Test execution details
  - `KeywordItem` — Recursive keyword display

## Development

### Backend development

```bash
go build -o robodiff
./robodiff /path/to/test/data
```

### Frontend development

```bash
cd web
npm install
npm run dev
```

The Vite dev server proxies `/api` requests to `localhost:8080` (see [web/vite.config.js](web/vite.config.js)), so run the Go backend separately:

```bash
./robodiff --addr :8080 /path/to/test/data
```

### Production build

```bash
cd web
npm run build
cd ..
go build -o robodiff
```

The Go binary serves the built React app from `web/dist/`. If `web/dist/index.html` is missing, the server returns a helpful message with build instructions.

## Project structure

```
├── main.go                  # CLI entrypoint
├── backend/
│   ├── server/
│   │   ├── server.go       # HTTP server and routes
│   │   └── store.go        # Folder scanner and run storage
│   └── diff/
│       ├── robot.go        # XML structure definitions
│       ├── parse.go        # XML parsing
│       ├── diff.go         # Comparison logic
│       └── report.go       # JSON diff payload builder
├── web/
│   ├── src/
│   │   ├── App.jsx         # Main application
│   │   ├── components/     # React components
│   │   └── utils/          # Helper functions
│   ├── dist/               # Built assets (gitignored)
│   └── package.json
└── AGENTS.md              # Contributor guide
```

## Testing

- **Frontend**: Run React development server with `npm run dev`
- **Backend**: Go tests can be added to `*_test.go` files (run with `go test ./...`)
- **Integration**: Start backend, open browser to http://localhost:8080

## Contributing

See [AGENTS.md](AGENTS.md) for detailed contributor guidelines including:

- Project structure and module organization
- Coding style and naming conventions
- Commit and pull request guidelines
- Security considerations

Contributions welcome. Open issues for bugs/feature requests. For code changes, please open a pull request.

## Compatibility / build

- Requires Go 1.21+ (module: `robot_diff`)
- Requires Node.js for frontend development (React 19, Vite)
- Backend uses only Go standard library (no external dependencies)

## License

Apache-2.0

This repository is a derivative of the Robot Framework Robot Diff tool (Nokia) and retains attribution to the original authors.
