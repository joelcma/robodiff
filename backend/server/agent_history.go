package backend

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"robot_diff/backend/store"
)

const agentMatching = "Tests are matched by suite path and test name; the nth test with a repeated name matches the nth. Only runs of the same project are used. A renamed or moved test appears as ABSENT/added/removed. Test IDs belong to the run they came from."

func agentRunRef(info store.RunInfo) map[string]any {
	return map[string]any{"id": info.ID, "name": agentLabel(info.Name), "relPath": agentLabel(info.RelPath), "project": agentLabel(info.Project), "modTime": info.ModTime}
}

// agentRunContext finds a run and the earlier runs of the same project,
// newest first. newer counts later runs, so callers can say the run is not
// the latest.
func (s *Server) agentRunContext(runID string) (current store.RunInfo, earlier []store.RunInfo, newer int, ok bool) {
	runs := s.store.ListRuns()
	for _, r := range runs {
		if r.ID == runID {
			current, ok = r, true
			break
		}
	}
	if !ok {
		return current, nil, 0, false
	}
	for _, r := range runs {
		if r.ID == runID || r.Project != current.Project {
			continue
		}
		switch {
		case r.ModTime.Before(current.ModTime):
			earlier = append(earlier, r)
		case r.ModTime.After(current.ModTime):
			newer++
		}
	}
	// ListRuns is newest first already; keep that explicit.
	sort.SliceStable(earlier, func(i, j int) bool { return earlier[i].ModTime.After(earlier[j].ModTime) })
	return current, earlier, newer, true
}

// outcomes of the run already parsed for the agent index, in the same shape
// as store.TestOutcomes for earlier runs.
func (a *agentIndex) outcomes() []store.TestOutcome {
	out := make([]store.TestOutcome, 0, len(a.testOrder))
	for _, id := range a.testOrder {
		t := a.tests[id]
		out = append(out, store.TestOutcome{ID: id, Name: t.name, Status: t.test.Status.Status, Message: t.test.Status.Message, Start: t.test.Status.StartTime})
	}
	return out
}

// outcomeKeys pairs each test with a cross-run key: its name plus how many
// earlier tests in the run share that name.
func outcomeKeys(tests []store.TestOutcome) []string {
	seen := make(map[string]int, len(tests))
	keys := make([]string, len(tests))
	for i, t := range tests {
		keys[i] = fmt.Sprintf("%s\x00%d", t.Name, seen[t.Name])
		seen[t.Name]++
	}
	return keys
}

func outcomesByKey(tests []store.TestOutcome) map[string]*store.TestOutcome {
	keys := outcomeKeys(tests)
	m := make(map[string]*store.TestOutcome, len(tests))
	for i := range tests {
		m[keys[i]] = &tests[i]
	}
	return m
}

func isFail(status string) bool { return strings.EqualFold(status, "FAIL") }

func sameFailure(a, b string) bool {
	return store.MessageSignature(a) == store.MessageSignature(b)
}

type outcomePair struct {
	current, base *store.TestOutcome
}

type outcomeComparison struct {
	newFailures, stillFailing, fixed, added, removed []outcomePair
}

func compareOutcomes(current, base []store.TestOutcome) outcomeComparison {
	var c outcomeComparison
	baseByKey := outcomesByKey(base)
	currentKeys := outcomeKeys(current)
	seen := make(map[string]bool, len(current))
	for i := range current {
		cur := &current[i]
		key := currentKeys[i]
		seen[key] = true
		b := baseByKey[key]
		switch {
		case b == nil:
			c.added = append(c.added, outcomePair{cur, nil})
			if isFail(cur.Status) {
				c.newFailures = append(c.newFailures, outcomePair{cur, nil})
			}
		case isFail(cur.Status) && isFail(b.Status):
			c.stillFailing = append(c.stillFailing, outcomePair{cur, b})
		case isFail(cur.Status):
			c.newFailures = append(c.newFailures, outcomePair{cur, b})
		case isFail(b.Status):
			c.fixed = append(c.fixed, outcomePair{cur, b})
		}
	}
	baseKeys := outcomeKeys(base)
	for i := range base {
		if !seen[baseKeys[i]] {
			c.removed = append(c.removed, outcomePair{nil, &base[i]})
		}
	}
	return c
}

func (c outcomeComparison) counts() map[string]int {
	return map[string]int{"newFailures": len(c.newFailures), "stillFailing": len(c.stillFailing), "fixed": len(c.fixed), "addedTests": len(c.added), "removedTests": len(c.removed)}
}

func outcomeUnavailable(err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "still loading; retry shortly"
	}
	return "could not read run: " + err.Error()
}

func (s *Server) agentCompare(ctx context.Context, a *agentIndex, runID, baseID string, offset, limit int) (map[string]any, int, string) {
	current, earlier, _, ok := s.agentRunContext(runID)
	if !ok {
		return nil, 404, "run not found"
	}
	var base store.RunInfo
	switch {
	case baseID == "" && len(earlier) == 0:
		return map[string]any{"current": agentRunRef(current), "base": nil, "note": "No earlier run of this project was found."}, 0, ""
	case baseID == "":
		base = earlier[0]
	case baseID == runID:
		return nil, 400, "base must be a different run"
	default:
		found := false
		for _, r := range s.store.ListRuns() {
			if r.ID == baseID {
				base, found = r, true
				break
			}
		}
		if !found {
			return nil, 404, "base run not found"
		}
	}
	baseTests, err := s.store.TestOutcomes(ctx, base.ID)
	if err != nil {
		return map[string]any{"current": agentRunRef(current), "base": agentRunRef(base), "unavailable": outcomeUnavailable(err)}, 0, ""
	}
	c := compareOutcomes(a.outcomes(), baseTests)
	item := func(p outcomePair) map[string]any {
		m := map[string]any{}
		if p.current != nil {
			m["test"] = a.testRef(p.current.ID)
		} else {
			m["name"] = agentLabel(p.base.Name)
		}
		if p.base != nil {
			m["baseTestId"] = p.base.ID
			m["baseStatus"] = p.base.Status
		} else {
			m["baseStatus"] = "ABSENT"
		}
		if p.current != nil && p.base != nil && isFail(p.current.Status) && isFail(p.base.Status) {
			m["sameFailure"] = sameFailure(p.current.Message, p.base.Message)
		}
		return m
	}
	page := func(pairs []outcomePair) map[string]any {
		items := make([]map[string]any, 0, len(pairs))
		for _, p := range pairs {
			items = append(items, item(p))
		}
		return agentPage(items, offset, limit)
	}
	return map[string]any{
		"current":      agentRunRef(current),
		"base":         agentRunRef(base),
		"counts":       c.counts(),
		"newFailures":  page(c.newFailures),
		"stillFailing": page(c.stillFailing),
		"fixed":        page(c.fixed),
		"addedTests":   page(c.added),
		"removedTests": page(c.removed),
		"matching":     agentMatching,
		"interpretation": "newFailures failed here but not in base (or are new tests). removedTests ran in base but not here: a shrinking run can mean an aborted suite. " +
			"sameFailure compares whitespace-normalized message previews; it is a hint, not proof of the same cause.",
	}, 0, ""
}

// agentPackComparison is the compact comparison with the previous run for
// analysis-pack. It never fails the pack: problems are reported inline.
func (s *Server) agentPackComparison(ctx context.Context, a *agentIndex, runID string) (map[string]any, map[string]bool) {
	_, earlier, _, ok := s.agentRunContext(runID)
	if !ok || len(earlier) == 0 {
		return map[string]any{"base": nil, "note": "No earlier run of this project was found."}, nil
	}
	base := earlier[0]
	baseTests, err := s.store.TestOutcomes(ctx, base.ID)
	if err != nil {
		return map[string]any{"base": agentRunRef(base), "unavailable": outcomeUnavailable(err)}, nil
	}
	c := compareOutcomes(a.outcomes(), baseTests)
	newIDs := make(map[string]bool, len(c.newFailures))
	for _, p := range c.newFailures {
		newIDs[p.current.ID] = true
	}
	counts := c.counts()
	delete(counts, "addedTests")
	return map[string]any{"base": agentRunRef(base), "counts": counts, "next": "Fetch /compare for the tests, or /tests/{testId}/history for one test."}, newIDs
}

type historyEntry struct {
	run   store.RunInfo
	tests map[string]*store.TestOutcome
	err   error
}

func (s *Server) agentHistory(ctx context.Context, a *agentIndex, runID, testID string, offset, limit, textLimit int) (map[string]any, int, string) {
	t := a.tests[testID]
	if t == nil {
		return nil, 404, "test not found"
	}
	current, earlier, newer, ok := s.agentRunContext(runID)
	if !ok {
		return nil, 404, "run not found"
	}
	cur := a.outcomes()
	keys := outcomeKeys(cur)
	var key string
	for i := range cur {
		if cur[i].ID == testID {
			key = keys[i]
			break
		}
	}

	start := minAgent(offset, len(earlier))
	window := earlier[start:minAgent(start+limit, len(earlier))]
	entries := make([]historyEntry, len(window))
	var wg sync.WaitGroup
	for i, r := range window {
		wg.Add(1)
		go func(i int, r store.RunInfo) {
			defer wg.Done()
			tests, err := s.store.TestOutcomes(ctx, r.ID)
			entries[i] = historyEntry{run: r, err: err}
			if err == nil {
				entries[i].tests = outcomesByKey(tests)
			}
		}(i, r)
	}
	wg.Wait()

	currentStatus := t.test.Status.Status
	items := make([]map[string]any, 0, len(entries))
	summary := map[string]any{"checked": len(entries)}
	countBy := map[string]int{"pass": 0, "fail": 0, "skip": 0, "other": 0, "absent": 0, "unavailable": 0}
	var lastPass any
	streak, streakOpen := 0, true
	changes, prevStatus := 0, currentStatus
	presentFails, present := 0, 0
	for _, e := range entries {
		item := map[string]any{"run": agentRunRef(e.run)}
		status := ""
		switch {
		case e.err != nil:
			status = "UNAVAILABLE"
			item["unavailable"] = outcomeUnavailable(e.err)
			countBy["unavailable"]++
		case e.tests[key] == nil:
			status = "ABSENT"
			countBy["absent"]++
		default:
			o := e.tests[key]
			status = o.Status
			item["testId"] = o.ID
			if !strings.EqualFold(o.Status, "PASS") {
				item["statusMessage"] = agentText(o.Message, 0, minAgent(textLimit, 256))
			}
			if isFail(o.Status) && isFail(currentStatus) {
				item["sameFailureAsCurrent"] = sameFailure(o.Message, t.test.Status.Message)
			}
			switch strings.ToLower(o.Status) {
			case "pass", "fail", "skip":
				countBy[strings.ToLower(o.Status)]++
			default:
				countBy["other"]++
			}
			present++
			if isFail(o.Status) {
				presentFails++
			}
			if lastPass == nil && strings.EqualFold(o.Status, "PASS") {
				lastPass = agentRunRef(e.run)
			}
			if !strings.EqualFold(o.Status, prevStatus) {
				changes++
			}
			prevStatus = o.Status
		}
		if streakOpen && strings.EqualFold(status, currentStatus) {
			streak++
		} else {
			streakOpen = false
		}
		item["status"] = status
		items = append(items, item)
	}
	for k, v := range countBy {
		summary[k] = v
	}
	summary["sameStatusStreak"] = streak
	summary["lastPass"] = lastPass
	summary["statusChanges"] = changes
	summary["pattern"] = historyPattern(currentStatus, present, presentFails)

	runsPage := agentPage(items, 0, len(items))
	runsPage["total"] = len(earlier)
	runsPage["offset"] = start
	runsPage["truncated"] = start > 0 || start+len(items) < len(earlier)
	runsPage["nextOffset"] = nil
	if start+len(items) < len(earlier) {
		runsPage["nextOffset"] = start + len(items)
	}
	return map[string]any{
		"test":          a.testRef(testID),
		"current":       map[string]any{"run": agentRunRef(current), "status": currentStatus},
		"newerRunCount": newer,
		"runs":          runsPage,
		"summary":       summary,
		"matching":      agentMatching,
		"interpretation": "runs are earlier runs, newest first. summary covers only the returned runs. pattern is a heuristic: " +
			"new-failure (no failure in the checked runs), consistently-failing, intermittent, no-history or not-failing. " +
			"sameStatusStreak counts consecutive earlier runs with the current status.",
	}, 0, ""
}

func historyPattern(currentStatus string, present, presentFails int) string {
	switch {
	case !isFail(currentStatus):
		return "not-failing"
	case present == 0:
		return "no-history"
	case presentFails == present:
		return "consistently-failing"
	case presentFails == 0:
		return "new-failure"
	default:
		return "intermittent"
	}
}
