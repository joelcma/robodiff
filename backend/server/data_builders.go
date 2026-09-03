package backend

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	rdiff "robot_diff/backend/diff"
)

type timeBreakdownNode struct {
	Name       string              `json:"name"`
	FullName   string              `json:"fullName"`
	Type       string              `json:"type"`
	Status     string              `json:"status"`
	DurationMs int64               `json:"durationMs"`
	TestCount  int                 `json:"testCount"`
	Children   []timeBreakdownNode `json:"children,omitempty"`
}

type timeBreakdownSummary struct {
	TotalDurationMs  int64   `json:"totalDurationMs"`
	SuiteCount       int     `json:"suiteCount"`
	TestCount        int     `json:"testCount"`
	LongestSuiteName string  `json:"longestSuiteName"`
	LongestSuiteMs   int64   `json:"longestSuiteMs"`
	LongestTestName  string  `json:"longestTestName"`
	LongestTestMs    int64   `json:"longestTestMs"`
	AccountedTestMs  int64   `json:"accountedTestMs"`
	AccountedPct     float64 `json:"accountedPct"`
}

type keywordTimingOccurrence struct {
	TestName   string `json:"testName"`
	DurationMs int64  `json:"durationMs"`
	Status     string `json:"status"`
}

type keywordTiming struct {
	Name              string                    `json:"name"`
	Owner             string                    `json:"owner,omitempty"`
	DisplayName       string                    `json:"displayName"`
	TotalDurationMs   int64                     `json:"totalDurationMs"`
	CallCount         int                       `json:"callCount"`
	AverageDurationMs int64                     `json:"averageDurationMs"`
	MaxDurationMs     int64                     `json:"maxDurationMs"`
	Occurrences       []keywordTimingOccurrence `json:"occurrences"`
}

func buildTestBodyKeywords(test *rdiff.Test) []rdiff.Keyword {
	if len(test.Body) > 0 {
		return orderedBodyToKeywords(test.Body)
	}
	keywords := make([]rdiff.Keyword, 0, len(test.Keywords)+len(test.Ifs)+len(test.Fors))
	keywords = append(keywords, test.Keywords...)
	keywords = append(keywords, controlFlowToKeywords(test.Ifs, test.Fors)...)
	return keywords
}

func orderedBodyToKeywords(body []rdiff.BodyItem) []rdiff.Keyword {
	out := make([]rdiff.Keyword, 0, len(body))
	for _, it := range body {
		switch {
		case it.Keyword != nil:
			out = append(out, *it.Keyword)
		case it.If != nil:
			out = append(out, ifToKeyword(*it.If))
		case it.For != nil:
			out = append(out, forToKeyword(*it.For))
		}
	}
	return out
}

func controlFlowToKeywords(ifs []rdiff.If, fors []rdiff.For) []rdiff.Keyword {
	out := make([]rdiff.Keyword, 0, len(ifs)+len(fors))
	for _, ifblk := range ifs {
		out = append(out, ifToKeyword(ifblk))
	}
	for _, forblk := range fors {
		out = append(out, forToKeyword(forblk))
	}
	return out
}

func ifToKeyword(ifblk rdiff.If) rdiff.Keyword {
	kw := rdiff.Keyword{
		Name:   "IF",
		Type:   "IF",
		Status: ifblk.Status,
	}
	children := make([]rdiff.Keyword, 0, len(ifblk.Branches))
	for _, br := range ifblk.Branches {
		name := strings.TrimSpace(br.Type)
		if br.Condition != "" {
			name = name + " " + br.Condition
		}
		bkw := rdiff.Keyword{
			Name:   name,
			Type:   "BRANCH",
			Status: br.Status,
		}
		if len(br.Body) > 0 {
			bkw.Keywords = append(bkw.Keywords, orderedBodyToKeywords(br.Body)...)
		} else {
			bkw.Keywords = append(bkw.Keywords, br.Keywords...)
			bkw.Keywords = append(bkw.Keywords, controlFlowToKeywords(br.Ifs, br.Fors)...)
		}
		if br.Return != nil {
			bkw.Keywords = append(bkw.Keywords, returnToKeyword(*br.Return))
		}
		children = append(children, bkw)
	}
	kw.Keywords = children
	return kw
}

func forToKeyword(forblk rdiff.For) rdiff.Keyword {
	name := "FOR"
	if len(forblk.Var) > 0 {
		name = name + " " + strings.Join(forblk.Var, ", ")
	}
	if forblk.Flavor != "" {
		name = name + " " + forblk.Flavor
	}
	if len(forblk.Value) > 0 {
		name = name + " " + strings.Join(forblk.Value, ", ")
	}

	kw := rdiff.Keyword{
		Name:   name,
		Type:   "FOR",
		Status: forblk.Status,
	}

	children := make([]rdiff.Keyword, 0, len(forblk.Iter))
	for i, it := range forblk.Iter {
		iterKw := rdiff.Keyword{
			Name:   fmt.Sprintf("ITER %d", i+1),
			Type:   "ITER",
			Status: it.Status,
		}
		if len(it.Body) > 0 {
			iterKw.Keywords = append(iterKw.Keywords, orderedBodyToKeywords(it.Body)...)
		} else {
			iterKw.Keywords = append(iterKw.Keywords, it.Keywords...)
			iterKw.Keywords = append(iterKw.Keywords, controlFlowToKeywords(it.Ifs, it.Fors)...)
		}
		if it.Return != nil {
			iterKw.Keywords = append(iterKw.Keywords, returnToKeyword(*it.Return))
		}
		children = append(children, iterKw)
	}
	kw.Keywords = children
	return kw
}

func returnToKeyword(ret rdiff.Return) rdiff.Keyword {
	args := make([]string, 0, len(ret.Value))
	for _, v := range ret.Value {
		args = append(args, fmt.Sprintf("value=%s", v))
	}
	return rdiff.Keyword{
		Name:      "RETURN",
		Type:      "RETURN",
		Arguments: args,
		Status:    ret.Status,
	}
}

func buildKeywordsData(keywords []rdiff.Keyword) []map[string]any {
	result := make([]map[string]any, len(keywords))
	for i, kw := range keywords {
		children := keywordChildrenInOrder(kw)

		result[i] = map[string]any{
			"name":          kw.Name,
			"type":          kw.Type,
			"status":        kw.Status.Status,
			"statusMessage": strings.TrimSpace(kw.Status.Message),
			"start":         kw.Status.StartTime,
			"end":           kw.Status.EndTime,
			"arguments":     kw.Arguments,
			"keywords":      buildKeywordsData(children),
			"messages":      buildMessagesData(kw.Messages),
		}
	}
	return result
}

func keywordChildrenInOrder(kw rdiff.Keyword) []rdiff.Keyword {
	if len(kw.Body) > 0 {
		return orderedBodyToKeywords(kw.Body)
	}
	children := make([]rdiff.Keyword, 0, len(kw.Keywords)+len(kw.Ifs)+len(kw.Fors))
	children = append(children, kw.Keywords...)
	children = append(children, controlFlowToKeywords(kw.Ifs, kw.Fors)...)
	return children
}

func buildMessagesData(messages []rdiff.Message) []map[string]any {
	result := make([]map[string]any, len(messages))
	for i, msg := range messages {
		result[i] = map[string]any{
			"level":     msg.Level,
			"timestamp": msg.Timestamp,
			"html":      msg.HTML,
			"text":      msg.Text,
		}
	}
	return result
}

func buildSuitesData(suite *rdiff.Suite) []map[string]any {
	var result []map[string]any

	// Add current suite if it has tests
	if len(suite.Tests) > 0 {
		tests := make([]map[string]any, len(suite.Tests))
		for i, test := range suite.Tests {
			tests[i] = map[string]any{
				"name":   test.Name,
				"status": test.Status.Status,
			}
		}
		result = append(result, map[string]any{
			"name":  suite.Name,
			"tests": tests,
		})
	}

	// Recursively add sub-suites
	for i := range suite.Suites {
		result = append(result, buildSuitesData(&suite.Suites[i])...)
	}

	return result
}

func buildTimeBreakdownData(suite *rdiff.Suite) (timeBreakdownNode, timeBreakdownSummary) {
	root := buildTimeBreakdownNode(suite, "")
	summary := timeBreakdownSummary{
		TotalDurationMs: root.DurationMs,
		SuiteCount:      countSuiteNodes(root) - 1,
		TestCount:       root.TestCount,
		AccountedTestMs: sumLeafTestDuration(root),
	}
	if summary.TotalDurationMs <= 0 {
		summary.TotalDurationMs = summary.AccountedTestMs
		root.DurationMs = summary.TotalDurationMs
	}
	if summary.TotalDurationMs > 0 && summary.AccountedTestMs > 0 {
		summary.AccountedPct = float64(summary.AccountedTestMs) / float64(summary.TotalDurationMs) * 100
	}
	longestSuiteName, longestSuiteMs := findLongestSuite(root, true)
	summary.LongestSuiteName = longestSuiteName
	summary.LongestSuiteMs = longestSuiteMs
	longestTestName, longestTestMs := findLongestTest(root)
	summary.LongestTestName = longestTestName
	summary.LongestTestMs = longestTestMs
	return root, summary
}

func buildKeywordTimingData(suite *rdiff.Suite) []keywordTiming {
	byKeyword := make(map[string]*keywordTiming)

	var visitSuite func(current *rdiff.Suite, prefix string)
	visitSuite = func(current *rdiff.Suite, prefix string) {
		fullName := current.Name
		if prefix != "" {
			fullName = prefix + "." + current.Name
		}

		for i := range current.Tests {
			test := &current.Tests[i]
			testName := fullName + "." + test.Name
			for _, keyword := range buildTestBodyKeywords(test) {
				collectKeywordTiming(byKeyword, keyword, testName)
			}
		}
		for i := range current.Suites {
			visitSuite(&current.Suites[i], fullName)
		}
	}

	visitSuite(suite, "")
	result := make([]keywordTiming, 0, len(byKeyword))
	for _, timing := range byKeyword {
		timing.AverageDurationMs = timing.TotalDurationMs / int64(timing.CallCount)
		sort.SliceStable(timing.Occurrences, func(i, j int) bool {
			return timing.Occurrences[i].DurationMs > timing.Occurrences[j].DurationMs
		})
		if len(timing.Occurrences) > 5 {
			timing.Occurrences = timing.Occurrences[:5]
		}
		result = append(result, *timing)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].TotalDurationMs == result[j].TotalDurationMs {
			return strings.ToLower(result[i].DisplayName) < strings.ToLower(result[j].DisplayName)
		}
		return result[i].TotalDurationMs > result[j].TotalDurationMs
	})
	return result
}

func collectKeywordTiming(byKeyword map[string]*keywordTiming, keyword rdiff.Keyword, testName string) {
	if !isSyntheticKeyword(keyword) {
		name := strings.TrimSpace(keyword.Name)
		owner := strings.TrimSpace(keyword.Owner)
		if name != "" {
			key := normalizeKeywordTimingName(owner) + "\x00" + normalizeKeywordTimingName(name)
			timing := byKeyword[key]
			if timing == nil {
				displayName := name
				if owner != "" {
					displayName = owner + "." + name
				}
				timing = &keywordTiming{
					Name:        name,
					Owner:       owner,
					DisplayName: displayName,
				}
				byKeyword[key] = timing
			}

			durationMs := durationMsFromStatus(keyword.Status)
			timing.TotalDurationMs += durationMs
			timing.CallCount++
			if durationMs > timing.MaxDurationMs {
				timing.MaxDurationMs = durationMs
			}
			timing.Occurrences = append(timing.Occurrences, keywordTimingOccurrence{
				TestName:   testName,
				DurationMs: durationMs,
				Status:     keyword.Status.Status,
			})
		}
	}

	for _, child := range keywordChildrenInOrder(keyword) {
		collectKeywordTiming(byKeyword, child, testName)
	}
}

func isSyntheticKeyword(keyword rdiff.Keyword) bool {
	switch keyword.Type {
	case "IF", "BRANCH", "FOR", "ITER", "RETURN":
		return true
	default:
		return false
	}
}

func normalizeKeywordTimingName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.ReplaceAll(value, "_", " ")), " "))
}

func buildTimeBreakdownNode(suite *rdiff.Suite, prefix string) timeBreakdownNode {
	fullName := suite.Name
	if prefix != "" {
		fullName = prefix + "." + suite.Name
	}

	children := make([]timeBreakdownNode, 0, len(suite.Suites)+len(suite.Tests))
	testCount := 0
	var childDurationMs int64

	for i := range suite.Suites {
		child := buildTimeBreakdownNode(&suite.Suites[i], fullName)
		children = append(children, child)
		testCount += child.TestCount
		childDurationMs += child.DurationMs
	}

	for _, test := range suite.Tests {
		durationMs := durationMsFromStatus(test.Status)
		children = append(children, timeBreakdownNode{
			Name:       test.Name,
			FullName:   fullName + "." + test.Name,
			Type:       "test",
			Status:     test.Status.Status,
			DurationMs: durationMs,
			TestCount:  1,
		})
		testCount += 1
		childDurationMs += durationMs
	}

	sort.SliceStable(children, func(i, j int) bool {
		if children[i].DurationMs == children[j].DurationMs {
			if children[i].Type == children[j].Type {
				return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
			}
			return children[i].Type == "suite"
		}
		return children[i].DurationMs > children[j].DurationMs
	})

	durationMs := durationMsFromStatus(suite.Status)
	if durationMs <= 0 {
		durationMs = childDurationMs
	}

	return timeBreakdownNode{
		Name:       suite.Name,
		FullName:   fullName,
		Type:       "suite",
		Status:     suite.Status.Status,
		DurationMs: durationMs,
		TestCount:  testCount,
		Children:   children,
	}
}

func durationMsFromStatus(status rdiff.Status) int64 {
	start, okStart := parseRobotTimestamp(status.StartTime)
	end, okEnd := parseRobotTimestamp(status.EndTime)
	if okStart && okEnd && !end.Before(start) {
		return end.Sub(start).Milliseconds()
	}
	if elapsedMs, ok := parseRobotElapsedMs(status.Elapsed); ok {
		return elapsedMs
	}
	return 0
}

func parseRobotTimestamp(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}

	layouts := []string{
		"20060102 15:04:05.000",
		"20060102 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func parseRobotElapsedMs(raw string) (int64, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false
	}

	seconds, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return int64(seconds * 1000), true
	}

	if d, err := time.ParseDuration(value); err == nil {
		return d.Milliseconds(), true
	}

	if d, err := time.ParseDuration(value + "s"); err == nil {
		return d.Milliseconds(), true
	}

	return 0, false
}

func countSuiteNodes(node timeBreakdownNode) int {
	if node.Type != "suite" {
		return 0
	}
	total := 1
	for _, child := range node.Children {
		total += countSuiteNodes(child)
	}
	return total
}

func sumLeafTestDuration(node timeBreakdownNode) int64 {
	if node.Type == "test" {
		return node.DurationMs
	}
	var total int64
	for _, child := range node.Children {
		total += sumLeafTestDuration(child)
	}
	return total
}

func findLongestSuite(node timeBreakdownNode, skipRoot bool) (string, int64) {
	bestName := ""
	bestDuration := int64(0)
	if node.Type == "suite" && !skipRoot {
		bestName = node.FullName
		bestDuration = node.DurationMs
	}
	for _, child := range node.Children {
		childName, childDuration := findLongestSuite(child, false)
		if childDuration > bestDuration {
			bestName = childName
			bestDuration = childDuration
		}
	}
	return bestName, bestDuration
}

func findLongestTest(node timeBreakdownNode) (string, int64) {
	if node.Type == "test" {
		return node.FullName, node.DurationMs
	}
	bestName := ""
	bestDuration := int64(0)
	for _, child := range node.Children {
		childName, childDuration := findLongestTest(child)
		if childDuration > bestDuration {
			bestName = childName
			bestDuration = childDuration
		}
	}
	return bestName, bestDuration
}
