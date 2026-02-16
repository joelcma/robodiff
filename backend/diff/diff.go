package robodiff

import (
	"sort"
	"strings"
)

// DiffResults manages the comparison results.
type DiffResults struct {
	stats       map[string][]*ItemStatus
	columnNames []string
}

func NewDiffResults() *DiffResults {
	return &DiffResults{
		stats:       make(map[string][]*ItemStatus, 128),
		columnNames: make([]string, 0, 4),
	}
}

func (dr *DiffResults) AddParsedOutput(robot *Robot, column string) {
	dr.addSuite(&robot.Suite, "")
	dr.columnNames = append(dr.columnNames, column)

	// Add missing statuses for all rows.
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
		statuses = make([]*ItemStatus, len(dr.columnNames), len(dr.columnNames)+4)
		for i := 0; i < len(dr.columnNames); i++ {
			statuses[i] = &ItemStatus{Name: "N/A", Status: "not_available"}
		}
	}

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

	statuses = append(statuses, &ItemStatus{Name: statusUpper, Status: statusLower})
	dr.stats[normalizedName] = statuses
}

func (dr *DiffResults) Rows() []*RowStatus {
	names := make([]string, 0, len(dr.stats))
	for name := range dr.stats {
		names = append(names, name)
	}
	sort.Strings(names)

	hasChildren := make(map[string]bool)
	for _, name := range names {
		parts := strings.Split(name, ".")
		for i := 1; i < len(parts); i++ {
			parent := strings.Join(parts[:i], ".")
			hasChildren[parent] = true
		}
	}

	rows := make([]*RowStatus, 0, len(names))
	for _, name := range names {
		if !hasChildren[name] {
			rows = append(rows, NewRowStatus(name, dr.stats[name]))
			continue
		}

		hasLeafChildren := false
		for _, otherName := range names {
			if strings.HasPrefix(otherName, name+".") {
				remainder := strings.TrimPrefix(otherName, name+".")
				if !strings.Contains(remainder, ".") && !hasChildren[otherName] {
					hasLeafChildren = true
					break
				}
			}
		}
		if hasLeafChildren {
			rows = append(rows, NewRowStatus(name, dr.stats[name]))
		}
	}

	return rows
}

type ItemStatus struct {
	Name   string
	Status string
}

type RowStatus struct {
	Name     string
	statuses []*ItemStatus
}

func NewRowStatus(name string, statuses []*ItemStatus) *RowStatus {
	return &RowStatus{Name: name, statuses: statuses}
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

func (rs *RowStatus) Statuses() []*ItemStatus { return rs.statuses }
