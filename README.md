# Robot Diff - Go Edition

A fast, native Go implementation of the Robot Framework output diff tool.

## Features

- Compare 2+ Robot Framework XML output files
- Generates HTML diff reports highlighting test status differences
- **History tracking**: Save comparison results with tags and view trends over time
- Zero dependencies (uses only Go standard library)
- Fast startup and low memory usage
- Single binary distribution

## Building

```bash
go build -o robotdiff
```

## Usage

```bash
# Basic usage
./robotdiff output1.xml output2.xml output3.xml

# With custom names
./robotdiff --name Env1 --name Env2 smoke1.xml smoke2.xml

# Custom report name and title
./robotdiff -r my_report.html -t "My Diff Report" output1.xml output2.xml

# Enable history tracking
./robotdiff --enable-history output1.xml output2.xml
```

## History Feature

The history feature allows you to save comparison results and track test trends over time:

1. **Enable History**: Use the `--enable-history` flag when generating reports
2. **Save to History**: In the report:
   - Select which test run (column) you want to save from the dropdown
   - Enter a tag (e.g., "nightly", "release-1.0")
   - Click "Save to History"
3. **View Trends**: Click the "History" button and select a tag to see:
   - Pass/fail trends over time
   - Visual timeline of test results
   - Pass rate graphs

History is stored in browser localStorage and optionally in a JSON file (`robotdiff_history.json` by default).

**Note**: When you have multiple test runs being compared (e.g., test1.xml, test2.xml), you must select which specific run to save to history. This allows you to track individual environments or configurations over time.

## Options

- `-r, --report <file>` - HTML report file (default: robotdiff.html)
- `-n, --name <name>` - Custom names for test runs (use multiple times)
- `-t, --title <title>` - Title for the report (default: Test Run Diff Report)
- `--enable-history` - Enable history saving feature in the report
- `--history-file <file>` - Path to history storage file (default: robotdiff_history.json)
- `-h, --help` - Show help

## Differences from Python version

- Uses standard Go XML parser instead of Robot Framework library
- Slightly faster startup and execution
- No external dependencies required
- Single compiled binary

# License: Apache-2.0

This program is a derivative of the Robot Framework Robot Diff tool by Nokia, originally licensed under the Apache-2.0 License. The original tool can be found at:
https://robotframework.org/robotframework/2.1.2/tools/robotdiff.html
