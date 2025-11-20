package main

import (
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

//go:embed template/report.html
var htmlTemplate string

//go:embed template/styles.css
var cssTemplate string

//go:embed template/app.js
var jsTemplate string

const usage = `Diff Tool for Robot Framework Outputs

Usage:  robotdiff [options] input_files

This script compares two or more Robot Framework output files and creates a
report where possible differences between test case statuses in each file
are highlighted. Main use case is verifying that results from executing same
test cases in different environments are same.

Options:
 -r --report file         HTML report file (created from the input files).
                          Default is 'robotdiff.html'.
 -n --name name           Custom names for test runs. If this option is used,
                          it must be used as many times as there are input
                          files. By default test run names are got from the
                          input file names.
 -t --title title         Title for the generated diff report. The default
                          title is 'Test Run Diff Report'.
 -h --help                Print this usage instruction.

Examples:
$ robotdiff output1.xml output2.xml output3.xml
$ robotdiff --name Env1 --name Env2 smoke1.xml smoke2.xml
`

type StringSlice []string

func (s *StringSlice) String() string {
	return strings.Join(*s, ", ")
}

func (s *StringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type Config struct {
	Report string
	Names  StringSlice
	Title  string
	Help   bool
}

func main() {
	config := parseArgs()

	if config.Help {
		fmt.Print(usage)
		os.Exit(0)
	}

	inputFiles := flag.Args()
	if len(inputFiles) < 2 {
		fmt.Fprintln(os.Stderr, "Error: At least 2 input files are required")
		fmt.Print(usage)
		os.Exit(1)
	}

	names := getNames(config.Names, inputFiles)
	if names == nil {
		os.Exit(1)
	}

	results := NewDiffResults()
	
	// Parse files in parallel
	type parseResult struct {
		index int
		robot *Robot
		err   error
	}
	
	resultChan := make(chan parseResult, len(inputFiles))
	var wg sync.WaitGroup
	
	for i, path := range inputFiles {
		wg.Add(1)
		go func(idx int, p string) {
			defer wg.Done()
			data, err := os.ReadFile(p)
			if err != nil {
				resultChan <- parseResult{idx, nil, fmt.Errorf("failed to read file: %w", err)}
				return
			}
			
			var robot Robot
			if err := xml.Unmarshal(data, &robot); err != nil {
				resultChan <- parseResult{idx, nil, fmt.Errorf("failed to parse XML: %w", err)}
				return
			}
			resultChan <- parseResult{idx, &robot, nil}
		}(i, path)
	}
	
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results in order
	parsedResults := make([]*Robot, len(inputFiles))
	for result := range resultChan {
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", inputFiles[result.index], result.err)
			os.Exit(1)
		}
		parsedResults[result.index] = result.robot
	}
	
	// Add results sequentially to maintain order
	for i, robot := range parsedResults {
		results.AddParsedOutput(robot, names[i])
	}

	reporter := NewDiffReporter(config.Report, config.Title, names)
	if err := reporter.Report(results); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Report: %s\n", reporter.OutPath)
}

func parseArgs() *Config {
	config := &Config{}

	flag.StringVar(&config.Report, "r", "robotdiff.html", "HTML report file")
	flag.StringVar(&config.Report, "report", "robotdiff.html", "HTML report file")
	flag.Var(&config.Names, "n", "Custom name for test run (can be used multiple times)")
	flag.Var(&config.Names, "name", "Custom name for test run (can be used multiple times)")
	flag.StringVar(&config.Title, "t", "Test Run Diff Report", "Title for the diff report")
	flag.StringVar(&config.Title, "title", "Test Run Diff Report", "Title for the diff report")
	flag.BoolVar(&config.Help, "h", false, "Show help")
	flag.BoolVar(&config.Help, "help", false, "Show help")

	flag.Usage = func() {
		fmt.Print(usage)
	}

	flag.Parse()
	return config
}

func getNames(names []string, paths []string) []string {
	if len(names) == 0 {
		return paths
	}
	if len(names) == len(paths) {
		return names
	}
	fmt.Fprintf(os.Stderr, "Different number of test run names (%d) and input files (%d).\n",
		len(names), len(paths))
	return nil
}

// Robot Framework XML structures
type Robot struct {
	XMLName xml.Name `xml:"robot"`
	Suite   Suite    `xml:"suite"`
}

type Suite struct {
	Name   string   `xml:"name,attr"`
	Suites []Suite  `xml:"suite"`
	Tests  []Test   `xml:"test"`
	Status Status   `xml:"status"`
}

type Test struct {
	Name   string `xml:"name,attr"`
	Status Status `xml:"status"`
}

type Status struct {
	Status string `xml:"status,attr"`
}

// DiffResults manages the comparison results
type DiffResults struct {
	stats       map[string][]*ItemStatus
	columnNames []string
}

func NewDiffResults() *DiffResults {
	return &DiffResults{
		stats:       make(map[string][]*ItemStatus, 128), // Pre-allocate for typical test count
		columnNames: make([]string, 0, 4), // Most comparisons are 2-4 files
	}
}

func (dr *DiffResults) AddParsedOutput(robot *Robot, column string) {
	dr.addSuite(&robot.Suite, "")
	dr.columnNames = append(dr.columnNames, column)

	// Add missing statuses for all rows
	for name, statuses := range dr.stats {
		for len(statuses) < len(dr.columnNames) {
			statuses = append(statuses, &ItemStatus{Name: "N/A", Status: "not_available"})
		}
		dr.stats[name] = statuses
	}
}

func (dr *DiffResults) addSuite(suite *Suite, parent string) {
	longname := suite.Name
	if parent != "" {
		longname = parent + "." + suite.Name
	}

	dr.addToStats(longname, suite.Status.Status)

	for i := range suite.Suites {
		dr.addSuite(&suite.Suites[i], longname)
	}

	for _, test := range suite.Tests {
		testLongname := longname + "." + test.Name
		dr.addToStats(testLongname, test.Status.Status)
	}
}

func (dr *DiffResults) addToStats(name, status string) {
	normalizedName := strings.ToLower(name)
	statuses, exists := dr.stats[normalizedName]

	if !exists {
		// Pre-allocate with capacity
		statuses = make([]*ItemStatus, len(dr.columnNames), len(dr.columnNames)+4)
		// Add missing statuses for previous columns
		for i := 0; i < len(dr.columnNames); i++ {
			statuses[i] = &ItemStatus{Name: "N/A", Status: "not_available"}
		}
	}

	// Avoid repeated string operations - cache common values
	var statusUpper, statusLower string
	switch status {
	case "PASS", "pass":
		statusUpper, statusLower = "PASS", "pass"
	case "FAIL", "fail":
		statusUpper, statusLower = "FAIL", "fail"
	default:
		statusUpper = strings.ToUpper(status)
		statusLower = strings.ToLower(status)
	}

	statuses = append(statuses, &ItemStatus{
		Name:   statusUpper,
		Status: statusLower,
	})

	dr.stats[normalizedName] = statuses
}

func (dr *DiffResults) Rows() []*RowStatus {
	// Pre-allocate exact size
	names := make([]string, 0, len(dr.stats))

	for name := range dr.stats {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build a map to track which items have children
	hasChildren := make(map[string]bool)
	childrenCount := make(map[string]int)
	
	for _, name := range names {
		parts := strings.Split(name, ".")
		for i := 1; i < len(parts); i++ {
			parent := strings.Join(parts[:i], ".")
			hasChildren[parent] = true
			childrenCount[parent]++
		}
	}

	rows := make([]*RowStatus, 0, len(names))
	for _, name := range names {
		// A row should be included if:
		// 1. It has no children (it's a test/leaf node), OR
		// 2. It has children that have no children (it's a suite with direct test children)
		
		if !hasChildren[name] {
			// This is a leaf node (test) - include it
			rows = append(rows, NewRowStatus(name, dr.stats[name]))
		} else {
			// This has children - check if any of its direct children are leaf nodes
			hasLeafChildren := false
			for _, otherName := range names {
				if strings.HasPrefix(otherName, name+".") {
					// Check if it's a direct child
					remainder := strings.TrimPrefix(otherName, name+".")
					if !strings.Contains(remainder, ".") {
						// It's a direct child - check if it's a leaf
						if !hasChildren[otherName] {
							hasLeafChildren = true
							break
						}
					}
				}
			}
			
			// Only include if it has direct leaf children (actual test suite)
			if hasLeafChildren {
				rows = append(rows, NewRowStatus(name, dr.stats[name]))
			}
		}
	}

	return rows
}

// ItemStatus represents the status of a single item in one run
type ItemStatus struct {
	Name   string
	Status string
}

// RowStatus represents a single row in the diff report
type RowStatus struct {
	Name     string
	statuses []*ItemStatus
}

func NewRowStatus(name string, statuses []*ItemStatus) *RowStatus {
	return &RowStatus{
		Name:     name,
		statuses: statuses,
	}
}

func (rs *RowStatus) Status() string {
	passed := false
	failed := false
	missing := false

	for _, stat := range rs.statuses {
		if stat.Name == "PASS" {
			passed = true
		} else if stat.Name == "FAIL" {
			failed = true
		} else if stat.Name == "N/A" {
			missing = true
		}
	}

	if passed && failed {
		return "diff"
	}
	if missing {
		return "missing"
	}
	if passed {
		return "all_passed"
	}
	return "all_failed"
}

func (rs *RowStatus) Explanation() string {
	switch rs.Status() {
	case "all_passed":
		return "All passed"
	case "all_failed":
		return "All failed"
	case "missing":
		return "Missing items"
	case "diff":
		return "Different statuses"
	default:
		return ""
	}
}

func (rs *RowStatus) Statuses() []*ItemStatus {
	return rs.statuses
}

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
	Title   string      `json:"title"`
	Columns []string    `json:"columns"`
	Suites  []JSONSuite `json:"suites"`
}

// DiffReporter generates the HTML report
type DiffReporter struct {
	OutPath string
	title   string
	columns []string
}

func NewDiffReporter(outpath, title string, columns []string) *DiffReporter {
	if outpath == "" {
		outpath = "robotdiff.html"
	}
	absPath, _ := filepath.Abs(outpath)
	return &DiffReporter{
		OutPath: absPath,
		title:   title,
		columns: columns,
	}
}

func (dr *DiffReporter) Report(results *DiffResults) error {
	// Build JSON data structure
	jsonData := dr.buildJSONData(results)
	
	// Create HTML file with embedded JSON, CSS, and JS
	f, err := os.Create(dr.OutPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer f.Close()

	// Serialize JSON
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Build single-file HTML with embedded CSS and JS
	html := strings.ReplaceAll(htmlTemplate, "{{TITLE}}", dr.title)
	html = strings.ReplaceAll(html, "{{DATA}}", string(jsonBytes))
	html = strings.ReplaceAll(html, `<link rel="stylesheet" href="styles.css" />`, "<style>"+cssTemplate+"</style>")
	html = strings.ReplaceAll(html, `<script src="app.js"></script>`, "<script>"+jsTemplate+"</script>")
	
	_, err = f.WriteString(html)
	return err
}

func (dr *DiffReporter) buildJSONData(results *DiffResults) *JSONReport {
	// Build suite hierarchy from the filtered rows
	suiteMap := make(map[string]*JSONSuite)
	var suiteOrder []string
	
	rows := results.Rows()
	
	// Build a set of all row names to identify suites vs tests
	rowSet := make(map[string]bool)
	for _, row := range rows {
		rowSet[row.Name] = true
	}
	
	// Identify which rows are suites (have children in the row set)
	suiteNames := make(map[string]bool)
	for _, row := range rows {
		for _, otherRow := range rows {
			if otherRow.Name != row.Name && strings.HasPrefix(otherRow.Name, row.Name+".") {
				suiteNames[row.Name] = true
				break
			}
		}
	}
	
	// Process only test rows (not suite rows)
	for _, row := range rows {
		// Skip if this is a suite row
		if suiteNames[row.Name] {
			continue
		}
		
		// This is a test row - extract suite and test name
		lastDot := strings.LastIndex(row.Name, ".")
		if lastDot < 0 {
			continue // Skip root items
		}
		
		suiteName := row.Name[:lastDot]
		testName := row.Name[lastDot+1:]
		
		// Create suite if it doesn't exist
		if _, exists := suiteMap[suiteName]; !exists {
			suiteMap[suiteName] = &JSONSuite{
				Name:  suiteName,
				Tests: make([]JSONTest, 0),
			}
			suiteOrder = append(suiteOrder, suiteName)
		}
		
		// Add test to suite
		testResults := make([]string, len(row.Statuses()))
		for i, status := range row.Statuses() {
			if status.Name == "N/A" {
				testResults[i] = "MISSING"
			} else {
				testResults[i] = status.Name // "PASS" or "FAIL"
			}
		}
		
		suiteMap[suiteName].Tests = append(suiteMap[suiteName].Tests, JSONTest{
			Name:    testName,
			Results: testResults,
		})
	}
	
	// Build ordered suite list - only include suites that have tests
	suites := make([]JSONSuite, 0, len(suiteOrder))
	for _, name := range suiteOrder {
		suite := suiteMap[name]
		if len(suite.Tests) > 0 {
			suites = append(suites, *suite)
		}
	}
	
	return &JSONReport{
		Title:   dr.title,
		Columns: dr.columns,
		Suites:  suites,
	}
}

