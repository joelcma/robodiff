package backend

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func agentInt(r *http.Request, key string, fallback, max int) (int, bool) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback, true
	}
	n, err := strconv.Atoi(value)
	return n, err == nil && n >= 0 && n <= max
}

// This namespace intentionally exposes only reads. Existing UI mutation and
// HTTP replay handlers are not part of the agent contract.
func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	offset, ok1 := agentInt(r, "offset", 0, 1000000000)
	limit, ok2 := agentInt(r, "limit", 20, 100)
	textOffset, ok3 := agentInt(r, "textOffset", 0, 1000000000)
	textLimit, ok4 := agentInt(r, "textLimit", 512, 2000)
	if !ok1 || !ok2 || !ok3 || !ok4 || limit < 1 || textLimit < 1 {
		writeError(w, 400, "offset/textOffset must be nonnegative integers; limit must be 1..100 and textLimit must be 1..2000")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/agent/"), "/")
	if len(parts) == 1 && parts[0] == "runs" {
		runs := s.store.ListRuns()
		// Cap labels as well as collection size. IDs are authoritative.
		for i := range runs {
			runs[i].Name = agentLabel(runs[i].Name)
			runs[i].RelPath = agentLabel(runs[i].RelPath)
		}
		writeJSON(w, 200, map[string]any{"schemaVersion": 1, "runs": agentPage(runs, offset, limit)})
		return
	}
	if len(parts) < 3 || parts[0] != "runs" || parts[1] == "" {
		writeError(w, 404, "agent route not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	_, _, robots, err := s.store.GetRuns(ctx, []string{parts[1]})
	if err != nil {
		status, code, msg, detail := classifyError(err)
		writeErrorWithCode(w, status, code, msg, detail)
		return
	}
	a := s.agentIndexFor(robots[0])
	payload := map[string]any{"schemaVersion": 1, "runId": parts[1], "idScope": "IDs are stable only while the run artifact is unchanged"}
	switch {
	case len(parts) == 3 && parts[2] == "analysis-pack":
		// This is the normal first call for an agent. Keep it deliberately small:
		// one representative per group, with one compact terminal node.
		packLimit := minAgent(limit, 5)
		groups := make([]map[string]any, 0, packLimit)
		for _, g := range a.groups[:minAgent(packLimit, len(a.groups))] {
			group := map[string]any{
				"id":                g.ID,
				"kind":              g.Kind,
				"affectedTestCount": len(g.members),
				"evidenceNodeId":    g.EvidenceNodeID,
				"message":           agentText(g.message, 0, minAgent(textLimit, 256)),
			}
			if len(g.members) > 0 {
				group["representativeTest"] = a.testRef(g.members[0])
			}
			if nodeID := agentRepresentativeFailure(a, g); nodeID != "" {
				group["representativeFailure"] = a.nodeRef(nodeID)
			}
			groups = append(groups, group)
		}
		payload["compact"] = true
		payload["runStatus"] = robots[0].Suite.Status.Status
		payload["counts"] = a.counts
		payload["executionErrorCount"] = len(robots[0].Errors)
		payload["groupCount"] = len(a.groups)
		payload["groups"] = groups
		payload["groupsTruncated"] = len(a.groups) > len(groups)
		payload["next"] = "Fetch /triage or /groups/{groupId}/tests for expanded evidence."
	case len(parts) == 3 && parts[2] == "triage":
		payload["runStatus"] = robots[0].Suite.Status.Status
		payload["runStatusMessage"] = agentText(robots[0].Suite.Status.Message, textOffset, textLimit)
		payload["executionErrorCount"] = len(robots[0].Errors)
		groups := make([]map[string]any, 0, len(a.groups))
		for _, g := range a.groups {
			examples := make([]map[string]any, 0, 3)
			for _, id := range g.members[:minAgent(3, len(g.members))] {
				examples = append(examples, a.testRef(id))
			}
			groups = append(groups, map[string]any{"id": g.ID, "kind": g.Kind, "evidenceNodeId": g.EvidenceNodeID, "message": agentText(g.message, 0, textLimit), "affectedTestCount": len(g.members), "examples": examples})
		}
		payload["counts"] = a.counts
		payload["groups"] = agentPage(groups, offset, limit)
		payload["grouping"] = "Matching keyword paths and whitespace-normalized messages suggest a shared failure, not a proven root cause. Tests may belong to multiple groups. Suite attribution requires an explicit Robot propagation message."
		payload["parserLimitations"] = "Keyword evidence covers kw, IF and FOR; other control structures may be absent. A missing failure node does not mean no failure occurred."
	case len(parts) == 3 && parts[2] == "errors":
		errors := robots[0].Errors
		start := minAgent(offset, len(errors))
		end := minAgent(start+limit, len(errors))
		items := []map[string]any{}
		for i := start; i < end; i++ {
			m := errors[i]
			items = append(items, map[string]any{"index": i, "level": m.Level, "timestamp": m.Timestamp, "text": agentText(m.Text, textOffset, textLimit)})
		}
		page := agentPage(errors, offset, limit)
		page["items"] = items
		payload["errors"] = page
	case len(parts) == 3 && parts[2] == "tests":
		status := strings.ToUpper(r.URL.Query().Get("status"))
		if status != "" && status != "FAIL" && status != "PASS" && status != "SKIP" {
			writeError(w, 400, "status must be FAIL, PASS or SKIP")
			return
		}
		tests := []map[string]any{}
		for _, id := range a.testOrder {
			if status == "" || strings.EqualFold(a.tests[id].test.Status.Status, status) {
				tests = append(tests, a.testRef(id))
			}
		}
		payload["tests"] = agentPage(tests, offset, limit)
	case len(parts) == 5 && parts[2] == "groups" && parts[4] == "tests":
		g := a.groupByID[parts[3]]
		if g == nil {
			writeError(w, 404, "group not found")
			return
		}
		members := make([]map[string]any, 0, len(g.members))
		for _, id := range g.members {
			members = append(members, a.testRef(id))
		}
		payload["tests"] = agentPage(members, offset, limit)
	case len(parts) == 4 && parts[2] == "tests":
		t := a.tests[parts[3]]
		if t == nil {
			writeError(w, 404, "test not found")
			return
		}
		payload["test"] = a.testRef(t.id)
		payload["fullName"] = agentText(t.name, textOffset, textLimit)
		payload["robotId"] = t.test.ID
		payload["source"] = agentText(t.source, textOffset, textLimit)
		payload["line"] = t.test.Line
		payload["statusMessage"] = agentText(t.test.Status.Message, textOffset, textLimit)
		payload["start"] = t.test.Status.StartTime
		payload["end"] = t.test.Status.EndTime
		payload["elapsed"] = t.test.Status.Elapsed
		payload["groupIds"] = agentPage(t.groups, offset, limit)
		failures := make([]map[string]any, 0, len(t.failures))
		for _, id := range t.failures {
			failures = append(failures, a.nodeRef(id))
		}
		payload["failures"] = agentPage(failures, offset, limit)
		roots := make([]map[string]any, 0, len(t.roots))
		for _, id := range t.roots {
			roots = append(roots, a.nodeRef(id))
		}
		payload["roots"] = agentPage(roots, offset, limit)
	case len(parts) == 4 && parts[2] == "nodes":
		n := a.nodes[parts[3]]
		if n == nil {
			writeError(w, 404, "node not found")
			return
		}
		payload["node"] = a.nodeRef(n.id)
		payload["name"] = agentText(n.kw.Name, textOffset, textLimit)
		payload["owner"] = agentText(n.kw.Owner, textOffset, textLimit)
		payload["source"] = agentText(n.kw.Source, textOffset, textLimit)
		payload["line"] = n.kw.Line
		payload["statusMessage"] = agentText(n.kw.Status.Message, textOffset, textLimit)
		payload["start"] = n.kw.Status.StartTime
		payload["end"] = n.kw.Status.EndTime
		payload["elapsed"] = n.kw.Status.Elapsed
		// Slice first: only selected messages/arguments need text conversion.
		start := minAgent(offset, len(n.kw.Messages))
		end := minAgent(start+limit, len(n.kw.Messages))
		messages := []map[string]any{}
		for i := start; i < end; i++ {
			m := n.kw.Messages[i]
			messages = append(messages, map[string]any{"index": i, "level": m.Level, "timestamp": m.Timestamp, "html": m.HTML, "text": agentText(m.Text, textOffset, textLimit), "screenshots": agentScreenshots(parts[1], m.Text)})
		}
		mp := agentPage(n.kw.Messages, offset, limit)
		mp["items"] = messages
		payload["messages"] = mp
		start = minAgent(offset, len(n.kw.Arguments))
		end = minAgent(start+limit, len(n.kw.Arguments))
		args := []map[string]any{}
		for i := start; i < end; i++ {
			args = append(args, map[string]any{"index": i, "value": agentText(n.kw.Arguments[i], textOffset, textLimit)})
		}
		ap := agentPage(n.kw.Arguments, offset, limit)
		ap["items"] = args
		payload["arguments"] = ap
		children := make([]map[string]any, 0, len(n.children))
		for _, id := range n.children {
			children = append(children, a.nodeRef(id))
		}
		payload["children"] = agentPage(children, offset, limit)
		failures := []map[string]any{}
		for _, id := range a.terminalFailures([]string{n.id}) {
			failures = append(failures, a.nodeRef(id))
		}
		payload["failures"] = agentPage(failures, offset, limit)
	default:
		writeError(w, 404, "agent route not found")
		return
	}
	writeJSON(w, 200, payload)
}

var agentScreenshotPattern = regexp.MustCompile(`screenshots/[^\s"'<>]+\.(?:png|jpg|jpeg|gif|webp)`)

func agentScreenshots(runID, text string) []string {
	matches := agentScreenshotPattern.FindAllString(text, 10)
	refs := make([]string, 0, len(matches))
	for _, path := range matches {
		if strings.Contains(path, "..") || len(path) > 2048 {
			continue
		}
		refs = append(refs, "/api/run-file?runId="+url.QueryEscape(runID)+"&path="+url.QueryEscape(path))
	}
	return refs
}
