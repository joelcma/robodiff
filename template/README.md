# Report template — developer notes

This folder contains the report UI templates and a small test harness used when developing or editing the UI.

## Repository layout

```
robot_diff/
├── main.go                      # Go code — parses robot XML and builds JSON + final HTML
├── template/                    # Front-end templates embedded into the single-file report
│   ├── report.html              # HTML template with placeholders (TITLE, DATA, HISTORY, etc.)
│   ├── styles.css               # Styling used in the report
│   ├── app.js                   # JavaScript logic for rendering UI and history features
│   ├── app.test.js              # Small JS test harness for core functions
│   └── test.html                # Test runner page — run in browser to exercise tests
```

## Testing & dev workflow

- Quick browser-based JS tests: open `template/test.html` in your browser. This will run the assertions in `app.test.js` and print results to the console and page.
- Node.js / automated JS tests: `app.test.js` is a simple, zero-dependency test harness — you can adapt it for your preferred runner (mocha/jest) if you want CI.

## Key UI functions (short summary)

- `calculateTestStatus(results)` — decides a single test's summary state:

  - `diff` if both PASS and FAIL are present (highest priority)
  - `missing` if MISSING exists without PASS+FAIL conflict
  - `all_passed` if all PASS
  - `all_failed` otherwise

- `calculateSuiteStatus(suite)` — aggregates suite-level status from individual tests; `diff` takes precedence when any test is `diff`/`missing`.

## Filtering & search

- There are three filter modes: All / Differences only / Failed-only. Failed-only shows rows where at least one run reported `FAIL`.
- The search box performs a client-side substring search of test full names and display names.

## Embedding / build behavior

The Go code uses `//go:embed` to embed `report.html`, `styles.css` and `app.js` into the compiled binary (or used at build time) — the `DiffReporter` replaces placeholders with embedded CSS/JS and the generated JSON to create a single `robotdiff.html` file.

When editing the templates you can:

1. Edit `template/app.js`, `template/styles.css`, and `template/report.html`.
2. Rebuild the binary (or run the tool) to re-generate the final HTML:

```bash
go build -o robotdiff
./robotdiff <input1.xml> <input2.xml>
```

## Example DATA

Example structured data consumed by the app (useful when testing or mocking data):

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
          results: ["PASS", "FAIL"],
        },
        { name: "another test", results: ["PASS", "MISSING"] },
      ],
    },
  ],
};
```
