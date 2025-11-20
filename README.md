# Robot Diff - Go Edition

A fast, native Go implementation of the Robot Framework output diff tool.

## Features

- Compare 2+ Robot Framework XML output files
- Generates HTML diff reports highlighting test status differences
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
```

## Options

- `-r, --report <file>` - HTML report file (default: robotdiff.html)
- `-n, --name <name>` - Custom names for test runs (use multiple times)
- `-t, --title <title>` - Title for the report (default: Test Run Diff Report)
- `-h, --help` - Show help

## Differences from Python version

- Uses standard Go XML parser instead of Robot Framework library
- Slightly faster startup and execution
- No external dependencies required
- Single compiled binary

# License: Apache-2.0

This program is a derivative of the Robot Framework Robot Diff tool by Nokia, originally licensed under the Apache-2.0 License. The original tool can be found at:
https://robotframework.org/robotframework/2.1.2/tools/robotdiff.html
