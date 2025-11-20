# Robot Diff - Template Architecture

## Structure

The project now uses a clean separation of concerns:

```
robot_diff/
├── main.go                      # Go code for parsing XML and generating JSON
├── template/
│   ├── report.html             # HTML template with placeholders
│   ├── styles.css              # All CSS styling
│   ├── app.js                  # JavaScript logic for rendering and interactions
│   ├── app.test.js             # Unit tests for JavaScript functions
│   └── test.html               # Test runner page
```

## Testing JavaScript

To test the JavaScript logic:

```bash
# Open the test runner in a browser
open template/test.html
```

The tests verify:

- `calculateTestStatus()` - Correctly determines test status (diff, missing, all_passed, all_failed)
- `calculateSuiteStatus()` - Correctly aggregates test statuses for suites

## Key Functions

### `calculateTestStatus(results)`

Determines the status of a test based on its results across runs:

- **`diff`** - Has both PASS and FAIL (priority #1 - actual disagreement)
- **`missing`** - Has MISSING but no PASS+FAIL conflict (priority #2)
- **`all_passed`** - All results are PASS
- **`all_failed`** - All results are FAIL

### Filter Logic

- **Show All** - Shows everything
- **Differences Only** - Shows tests with `diff` or `missing` status
- **Failed Only** - Shows tests with `all_failed`, `diff`, or `missing` status

## Build Process

The Go code embeds the templates using `//go:embed` and generates a single HTML file with:

1. JSON data embedded in a `<script>` tag
2. CSS inlined in a `<style>` tag
3. JavaScript inlined in a `<script>` tag

This creates a completely self-contained report file that can be opened anywhere.

## Example Test Data

```javascript
const DATA = {
  title: "Test Run Diff Report",
  columns: ["Run 1", "Run 2"],
  suites: [
    {
      name: "01 route number",
      tests: [
        {
          name: "create routenumber with existing name should fail",
          results: ["PASS", "FAIL"], // Will show as "diff"
        },
        {
          name: "another test",
          results: ["PASS", "MISSING"], // Will show as "missing"
        },
      ],
    },
  ],
};
```
