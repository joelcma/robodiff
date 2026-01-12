package robotdiff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

type Templates struct {
	HTML string
	CSS  string
	JS   string
}

// DiffReporter generates the HTML report.
type DiffReporter struct {
	OutPath    string
	title      string
	columns    []string
	inputFiles []string
	templates  Templates
}

func NewDiffReporter(outpath, title string, columns []string, inputFiles []string, templates Templates) *DiffReporter {
	if outpath == "" {
		outpath = "robotdiff.html"
	}
	absPath, _ := filepath.Abs(outpath)
	return &DiffReporter{
		OutPath:    absPath,
		title:      title,
		columns:    columns,
		inputFiles: inputFiles,
		templates:  templates,
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

// History structures
type HistoryEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Tag       string      `json:"tag"`
	Title     string      `json:"title"`
	Columns   []string    `json:"columns"`
	Suites    []JSONSuite `json:"suites"`
}

type HistoryStore struct {
	Entries []HistoryEntry `json:"entries"`
}

func LoadHistory(path string) (*HistoryStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &HistoryStore{Entries: []HistoryEntry{}}, nil
		}
		return nil, err
	}

	var store HistoryStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return &store, nil
}

func (hs *HistoryStore) Save(path string) error {
	data, err := json.MarshalIndent(hs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (hs *HistoryStore) AddEntry(entry HistoryEntry) {
	hs.Entries = append(hs.Entries, entry)
	sort.Slice(hs.Entries, func(i, j int) bool {
		return hs.Entries[i].Timestamp.After(hs.Entries[j].Timestamp)
	})
}

func (hs *HistoryStore) GetByTag(tag string) []HistoryEntry {
	var results []HistoryEntry
	for _, entry := range hs.Entries {
		if entry.Tag == tag {
			results = append(results, entry)
		}
	}
	return results
}

func (hs *HistoryStore) GetAllTags() []string {
	tagSet := make(map[string]bool)
	for _, entry := range hs.Entries {
		tagSet[entry.Tag] = true
	}

	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func (dr *DiffReporter) Report(results *DiffResults, historyPath string, enableHistory bool) error {
	jsonData := dr.BuildJSONData(results)

	var historyData *HistoryStore
	if enableHistory && historyPath != "" {
		var err error
		historyData, err = LoadHistory(historyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load history: %v\n", err)
			historyData = &HistoryStore{Entries: []HistoryEntry{}}
		}
	}

	f, err := os.Create(dr.OutPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer f.Close()

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	historyBytes := []byte("null")
	if historyData != nil {
		historyBytes, err = json.Marshal(historyData)
		if err != nil {
			return fmt.Errorf("failed to marshal history: %w", err)
		}
	}

	html := strings.ReplaceAll(dr.templates.HTML, "{{TITLE}}", dr.title)
	html = strings.ReplaceAll(html, "{{DATA}}", string(jsonBytes))
	html = strings.ReplaceAll(html, "{{HISTORY}}", string(historyBytes))
	html = strings.ReplaceAll(html, "{{HISTORY_FILE}}", historyPath)
	html = strings.ReplaceAll(html, "{{HISTORY_ENABLED}}", fmt.Sprintf("%t", enableHistory))
	html = strings.ReplaceAll(html, `<link rel="stylesheet" href="styles.css" />`, "<style>"+dr.templates.CSS+"</style>")
	html = strings.ReplaceAll(html, `<script src="app.js"></script>`, "<script>"+dr.templates.JS+"</script>")

	_, err = f.WriteString(html)
	return err
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
