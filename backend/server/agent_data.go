package backend

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	rdiff "robot_diff/backend/diff"
)

// IDs describe positions in the parsed artifact, not names (which can repeat).
// They are stable while that artifact is unchanged, but are not cross-run IDs.
type agentNode struct {
	id, parent, testID, suiteID string
	kw                          rdiff.Keyword
	children                    []string
}
type agentTest struct {
	id, suiteID, name, source string
	test                      *rdiff.Test
	roots, failures, groups   []string
}
type agentGroup struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	EvidenceNodeID string `json:"evidenceNodeId,omitempty"`
	members        []string
	message        string
}
type agentIndex struct {
	nodes     map[string]*agentNode
	tests     map[string]*agentTest
	testOrder []string
	groups    []*agentGroup
	groupByID map[string]*agentGroup
	counts    map[string]int
}

// The store replaces parsed Robot pointers when files change. Retain just the
// most recently inspected index so successive drill-downs avoid walking the
// whole run again, without accumulating a second index for every run.
func (s *Server) agentIndexFor(robot *rdiff.Robot) *agentIndex {
	s.agentMu.Lock()
	defer s.agentMu.Unlock()
	if s.agentRobot != robot {
		s.agentData = newAgentIndex(&robot.Suite)
		s.agentRobot = robot
	}
	return s.agentData
}

func newAgentIndex(root *rdiff.Suite) *agentIndex {
	a := &agentIndex{nodes: map[string]*agentNode{}, tests: map[string]*agentTest{}, groupByID: map[string]*agentGroup{}, counts: map[string]int{"total": 0, "pass": 0, "fail": 0, "skip": 0, "other": 0}}
	a.addSuite(root, "s0", "", nil)
	sort.SliceStable(a.groups, func(i, j int) bool {
		left, right := a.groups[i], a.groups[j]
		lf, rf := strings.HasPrefix(left.Kind, "suite-"), strings.HasPrefix(right.Kind, "suite-")
		if lf != rf {
			return lf
		}
		return len(left.members) > len(right.members)
	})
	return a
}

func (a *agentIndex) addNodes(kws []rdiff.Keyword, prefix, parent, suiteID, testID string) []string {
	ids := make([]string, 0, len(kws))
	for i, kw := range kws {
		id := fmt.Sprintf("%s-k%d", prefix, i)
		n := &agentNode{id: id, parent: parent, suiteID: suiteID, testID: testID, kw: kw}
		a.nodes[id] = n
		n.children = a.addNodes(keywordChildrenInOrder(kw), id, id, suiteID, testID)
		ids = append(ids, id)
	}
	return ids
}

// Only follow FAIL ancestors. An unsuccessful attempt inside a successful retry
// is evidence in the tree, but is not a terminal failure of its test.
func (a *agentIndex) terminalFailures(ids []string) []string {
	var result []string
	for _, id := range ids {
		n := a.nodes[id]
		if !strings.EqualFold(n.kw.Status.Status, "FAIL") {
			continue
		}
		children := a.terminalFailures(n.children)
		if len(children) == 0 {
			result = append(result, id)
		} else {
			result = append(result, children...)
		}
	}
	return result
}

func (a *agentIndex) group(key, kind, nodeID, message string) *agentGroup {
	sum := sha256.Sum256([]byte(key))
	id := fmt.Sprintf("g%x", sum[:12])
	if g := a.groupByID[id]; g != nil {
		return g
	}
	g := &agentGroup{ID: id, Kind: kind, EvidenceNodeID: nodeID, message: message, members: []string{}}
	a.groups = append(a.groups, g)
	a.groupByID[id] = g
	return g
}

func (a *agentIndex) addSuite(s *rdiff.Suite, id, parentName string, inherited map[string]*agentGroup) {
	name := s.Name
	if parentName != "" {
		name = parentName + "." + name
	}
	fixtures := map[string]*agentGroup{}
	for k, v := range inherited {
		fixtures[k] = v
	}
	roots := a.addNodes(s.Keywords, id, "", id, "")
	for _, root := range roots {
		n := a.nodes[root]
		kind := strings.ToLower(n.kw.Type)
		if (kind != "setup" && kind != "teardown") || !strings.EqualFold(n.kw.Status.Status, "FAIL") {
			continue
		}
		g := a.group("fixture:"+root, "suite-"+kind, root, n.kw.Status.Message)
		fixtures[kind] = g
	}
	for i := range s.Tests {
		t := &s.Tests[i]
		tid := fmt.Sprintf("%s-t%d", id, i)
		at := &agentTest{id: tid, suiteID: id, name: name + "." + t.Name, source: s.Source, test: t, groups: []string{}}
		at.roots = a.addNodes(buildTestBodyKeywords(t), tid, "", id, tid)
		a.tests[tid] = at
		a.testOrder = append(a.testOrder, tid)
		a.counts["total"]++
		status := strings.ToLower(t.Status.Status)
		switch status {
		case "pass", "fail", "skip":
			a.counts[status]++
		default:
			a.counts["other"]++
		}
		if status != "fail" {
			continue
		}
		at.failures = a.terminalFailures(at.roots)
		// Robot's explicit propagation markers establish fixture attribution. Do not
		// assume every failure below a failed suite comes from its setup/teardown.
		lower := strings.ToLower(t.Status.Message)
		for _, kind := range []string{"setup", "teardown"} {
			if g := fixtures[kind]; g != nil && (strings.Contains(lower, "parent suite "+kind+" failed") || strings.Contains(lower, "suite "+kind+" failed")) {
				a.attach(at, g)
			}
		}
		for _, nid := range at.failures {
			n := a.nodes[nid]
			msg := n.kw.Status.Message
			if strings.TrimSpace(msg) == "" {
				msg = t.Status.Message
			}
			// Normalize whitespace only: replacing numbers/URLs can merge unrelated
			// assertion failures. Full original evidence remains available by node ID.
			var path []string
			for cur := n; cur != nil; cur = a.nodes[cur.parent] {
				path = append(path, cur.kw.Owner+"."+cur.kw.Name+":"+cur.kw.Type)
			}
			key := strings.Join(path, "\x00") + "\x01" + strings.Join(strings.Fields(msg), " ")
			a.attach(at, a.group(key, "test-failure", nid, msg))
		}
		if len(at.groups) == 0 {
			a.attach(at, a.group("status:"+strings.Join(strings.Fields(t.Status.Message), " "), "test-status", "", t.Status.Message))
		}
	}
	for i := range s.Suites {
		a.addSuite(&s.Suites[i], fmt.Sprintf("%s-s%d", id, i), name, fixtures)
	}
}

func (a *agentIndex) attach(t *agentTest, g *agentGroup) {
	for _, id := range t.groups {
		if id == g.ID {
			return
		}
	}
	t.groups = append(t.groups, g.ID)
	g.members = append(g.members, t.id)
}

// Text is independently pageable in Unicode code points, including arguments
// and status messages. Large request/response bodies never expand by default.
func agentText(s string, offset, limit int) map[string]any {
	runes := []rune(s)
	start := minAgent(offset, len(runes))
	end := minAgent(start+limit, len(runes))
	var next any
	if end < len(runes) {
		next = end
	}
	return map[string]any{"text": string(runes[start:end]), "totalChars": len(runes), "offset": start, "nextOffset": next, "truncated": start > 0 || end < len(runes)}
}
func minAgent(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func agentLabel(s string) string {
	r := []rune(s)
	if len(r) > 512 {
		return string(r[:512]) + "…"
	}
	return s
}
func agentPage[T any](items []T, offset, limit int) map[string]any {
	start := minAgent(offset, len(items))
	end := minAgent(start+limit, len(items))
	page := make([]T, end-start)
	copy(page, items[start:end])
	var next any
	if end < len(items) {
		next = end
	}
	return map[string]any{"items": page, "total": len(items), "offset": start, "nextOffset": next, "truncated": start > 0 || end < len(items)}
}
func (a *agentIndex) nodeRef(id string) map[string]any {
	n := a.nodes[id]
	return map[string]any{"id": id, "name": agentLabel(n.kw.Name), "owner": agentLabel(n.kw.Owner), "type": n.kw.Type, "status": n.kw.Status.Status, "parentId": n.parent, "testId": n.testID, "suiteId": n.suiteID}
}
func (a *agentIndex) testRef(id string) map[string]any {
	t := a.tests[id]
	return map[string]any{"id": id, "name": agentLabel(t.name), "status": t.test.Status.Status}
}

func agentRepresentativeFailure(a *agentIndex, g *agentGroup) string {
	if g.EvidenceNodeID == "" {
		return ""
	}
	failures := a.terminalFailures([]string{g.EvidenceNodeID})
	if len(failures) > 0 {
		return failures[0]
	}
	return g.EvidenceNodeID
}
