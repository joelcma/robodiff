package robodiff

import (
	"os"
	"path/filepath"
	"strings"
)

// JSON output structures
type JSONTest struct {
	Name    string   `json:"name"`
	Results []string `json:"results"`
}

type JSONSuite struct {
	Name  string     `json:"name"`
	Tests []JSONTest `json:"tests"`
}

type JSONReport struct {
	Title       string      `json:"title"`
	Columns     []string    `json:"columns"`
	ReportLinks []string    `json:"reportLinks"`
	Suites      []JSONSuite `json:"suites"`
}

// DiffReporter builds the JSON diff payload used by the server/React UI.
type DiffReporter struct {
	title      string
	columns    []string
	inputFiles []string
}

func NewDiffReporter(title string, columns []string, inputFiles []string) *DiffReporter {
	return &DiffReporter{
		title:      title,
		columns:    columns,
		inputFiles: inputFiles,
	}
}

func (dr *DiffReporter) detectReportLinks() []string {
	links := make([]string, len(dr.inputFiles))
	for i, inputFile := range dr.inputFiles {
		absPath, err := filepath.Abs(inputFile)
		if err != nil {
			links[i] = ""
			continue
		}

		dir := filepath.Dir(absPath)
		reportPath := filepath.Join(dir, "report.html")

		if _, err := os.Stat(reportPath); err == nil {
			links[i] = "file://" + reportPath
		} else {
			links[i] = ""
		}
	}
	return links
}

func (dr *DiffReporter) BuildJSONData(results *DiffResults) *JSONReport {
	suiteMap := make(map[string]*JSONSuite)
	var suiteOrder []string

	rows := results.Rows()
	reportLinks := dr.detectReportLinks()

	suiteNames := make(map[string]bool)
	for _, row := range rows {
		for _, otherRow := range rows {
			if otherRow.Name != row.Name && strings.HasPrefix(otherRow.Name, row.Name+".") {
				suiteNames[row.Name] = true
				break
			}
		}
	}

	for _, row := range rows {
		if suiteNames[row.Name] {
			continue
		}

		lastDot := strings.LastIndex(row.Name, ".")
		if lastDot < 0 {
			continue
		}

		suiteName := row.Name[:lastDot]
		testName := row.Name[lastDot+1:]

		if _, exists := suiteMap[suiteName]; !exists {
			suiteMap[suiteName] = &JSONSuite{Name: suiteName, Tests: make([]JSONTest, 0)}
			suiteOrder = append(suiteOrder, suiteName)
		}

		testResults := make([]string, len(row.Statuses()))
		for i, status := range row.Statuses() {
			if status.Name == "N/A" {
				testResults[i] = "MISSING"
			} else {
				testResults[i] = status.Name
			}
		}

		suiteMap[suiteName].Tests = append(suiteMap[suiteName].Tests, JSONTest{Name: testName, Results: testResults})
	}

	suites := make([]JSONSuite, 0, len(suiteOrder))
	for _, name := range suiteOrder {
		suite := suiteMap[name]
		if len(suite.Tests) > 0 {
			suites = append(suites, *suite)
		}
	}

	return &JSONReport{
		Title:       dr.title,
		Columns:     dr.columns,
		ReportLinks: reportLinks,
		Suites:      suites,
	}
}
