package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rdiff "robot_diff/backend/diff"
	"robot_diff/backend/store"
)

const agentFixture = `<robot>
<suite name="Root" source="/tests/root.robot" id="s1">
 <suite name="Broken setup">
  <kw name="Prepare" type="SETUP"><kw name="Connect" owner="DatabaseLibrary" source="/tests/db.resource" lineno="12"><msg level="FAIL">connection refused</msg><status status="FAIL">connection refused</status></kw><status status="FAIL">connection refused</status></kw>
  <test name="Duplicate" id="s1-s1-t1" lineno="4"><status status="FAIL">Parent suite setup failed: connection refused</status></test>
  <suite name="Nested"><test name="Duplicate"><status status="FAIL">Parent suite setup failed: connection refused</status></test></suite>
  <status status="FAIL"/>
 </suite>
 <suite name="Independent" source="/tests/independent.robot">
  <test name="Duplicate" id="s1-s2-t1" lineno="9">
   <kw name="Wait Until Keyword Succeeds"><kw name="Poll"><status status="FAIL">temporary</status></kw><kw name="Poll"><status status="PASS"/></kw><status status="PASS"/></kw>
   <kw name="Validate"><if><branch type="IF" condition="True"><kw name="Should Be Equal" owner="BuiltIn"><arg>actual</arg><msg level="INFO" time="2026-09-10T10:00:00" html="yes">&lt;img src="screenshots/example.png"&gt;</msg><status status="FAIL" start="2026-09-10T10:00:00" elapsed="0.1">actual != expected</status></kw><status status="FAIL"/></branch><status status="FAIL"/></if><status status="FAIL">actual != expected</status></kw>
   <status status="FAIL">actual != expected</status>
  </test>
  <test name="Duplicate"><kw name="Validate"><if><branch type="IF" condition="True"><kw name="Should Be Equal" owner="BuiltIn"><status status="FAIL"> actual   != expected </status></kw><status status="FAIL"/></branch><status status="FAIL"/></if><status status="FAIL"/></kw><status status="FAIL">actual != expected</status></test>
  <test name="Recovered"><kw name="Retry"><kw name="Attempt"><status status="FAIL">temporary</status></kw><status status="PASS"/></kw><status status="PASS"/></test>
  <test name="Skipped"><status status="SKIP">disabled</status></test>
  <status status="FAIL"/>
 </suite>
 <status status="FAIL"/>
</suite>
<errors><msg level="ERROR">Library import failed</msg></errors>
</robot>`

func TestAgentTriagePreservesFixtureEvidenceAndIgnoresRecoveredRetries(t *testing.T) {
	robot, err := rdiff.ParseRobotXMLBytes([]byte(agentFixture))
	if err != nil {
		t.Fatal(err)
	}
	a := newAgentIndex(&robot.Suite)
	if a.counts["total"] != 6 || a.counts["fail"] != 4 || a.counts["pass"] != 1 || a.counts["skip"] != 1 {
		t.Fatalf("counts: %v", a.counts)
	}
	if len(a.groups) != 2 {
		t.Fatalf("groups: %+v", a.groups)
	}
	setup, g := a.groups[0], a.groups[1]
	if setup.Kind != "suite-setup" || len(setup.members) != 2 || g.Kind != "test-failure" || len(g.members) != 2 {
		t.Fatalf("groups: %+v %+v", setup, g)
	}
	terminal := a.terminalFailures([]string{setup.EvidenceNodeID})
	if len(terminal) != 1 || a.nodes[terminal[0]].kw.Name != "Connect" {
		t.Fatalf("fixture leaves: %v", terminal)
	}
	n := a.nodes[terminal[0]]
	if n.kw.Source != "/tests/db.resource" || n.kw.Line != "12" {
		t.Fatalf("source lost: %+v", n.kw)
	}
	for _, id := range g.members {
		at := a.tests[id]
		if len(at.failures) != 1 || a.nodes[at.failures[0]].kw.Name != "Should Be Equal" {
			t.Fatalf("wrong failure leaves: %+v", at)
		}
	}
	if g.members[0] == g.members[1] {
		t.Fatal("duplicate names must have distinct IDs")
	}
	if len(robot.Errors) != 1 || robot.Errors[0].Text != "Library import failed" {
		t.Fatalf("execution errors: %+v", robot.Errors)
	}
}

func TestAgentTeardownDoesNotHideIndependentTestFailure(t *testing.T) {
	xml := `<robot><suite name="Root"><kw name="Cleanup" type="TEARDOWN"><status status="FAIL">cleanup error</status></kw>
 <test name="Both"><kw name="Assert"><status status="FAIL">bad value</status></kw><status status="FAIL">bad value
 Also parent suite teardown failed: cleanup error</status></test>
 <test name="Independent"><kw name="Assert"><status status="FAIL">different value</status></kw><status status="FAIL">different value</status></test>
 <status status="FAIL"/></suite></robot>`
	robot, err := rdiff.ParseRobotXMLBytes([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	a := newAgentIndex(&robot.Suite)
	if len(a.groups) != 3 || a.groups[0].Kind != "suite-teardown" || len(a.groups[0].members) != 1 {
		t.Fatalf("wrong grouping: %+v", a.groups)
	}
	if len(a.tests["s0-t0"].groups) != 2 || len(a.tests["s0-t1"].groups) != 1 {
		t.Fatal("independent test attribution lost")
	}
}

func agentTestServer(t *testing.T, xml string) (*http.ServeMux, string) {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "output.xml"), []byte(xml))
	rs := store.NewRunStore(root, time.Hour)
	rs.ScanOnce()
	runs := rs.ListRuns()
	if len(runs) != 1 {
		t.Fatalf("runs: %v", runs)
	}
	mux := http.NewServeMux()
	NewServer("", rs).registerRoutes(mux)
	return mux, runs[0].ID
}
func agentGet(t *testing.T, mux http.Handler, path string) map[string]any {
	t.Helper()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", path, nil))
	if rr.Code != 200 {
		t.Fatalf("GET %s: %d %s", path, rr.Code, rr.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	return data
}
func TestAgentHTTPDrillDownAndPagination(t *testing.T) {
	mux, id := agentTestServer(t, agentFixture)
	base := "/api/agent/runs/" + id
	pack := agentGet(t, mux, base+"/analysis-pack")
	if pack["compact"] != true || pack["groupCount"] != float64(2) || pack["executionErrorCount"] != float64(1) {
		t.Fatalf("analysis pack summary: %v", pack)
	}
	packGroups := pack["groups"].([]any)
	if len(packGroups) != 2 {
		t.Fatalf("analysis pack groups: %v", packGroups)
	}
	firstPackGroup := packGroups[0].(map[string]any)
	if firstPackGroup["representativeTest"] == nil || firstPackGroup["representativeFailure"] == nil {
		t.Fatalf("analysis pack evidence: %v", firstPackGroup)
	}
	runs := agentGet(t, mux, "/api/agent/runs?limit=1")["runs"].(map[string]any)
	if runs["total"] != float64(1) {
		t.Fatalf("runs: %v", runs)
	}
	triage := agentGet(t, mux, base+"/triage?limit=1")
	groups := triage["groups"].(map[string]any)
	if groups["total"] != float64(2) || groups["nextOffset"] != float64(1) || triage["executionErrorCount"] != float64(1) {
		t.Fatalf("triage: %v", triage)
	}
	group := groups["items"].([]any)[0].(map[string]any)
	members := agentGet(t, mux, base+"/groups/"+group["id"].(string)+"/tests?limit=1")["tests"].(map[string]any)
	if members["total"] != float64(2) {
		t.Fatalf("members: %v", members)
	}
	node := agentGet(t, mux, base+"/nodes/"+group["evidenceNodeId"].(string))
	leaf := node["failures"].(map[string]any)["items"].([]any)[0].(map[string]any)
	evidence := agentGet(t, mux, base+"/nodes/"+leaf["id"].(string))
	if evidence["line"] != "12" {
		t.Fatalf("evidence: %v", evidence)
	}
	test := agentGet(t, mux, base+"/tests/s0-s1-t0")
	if test["robotId"] != "s1-s2-t1" || test["line"] != "9" {
		t.Fatalf("test metadata: %v", test)
	}
	failure := test["failures"].(map[string]any)["items"].([]any)[0].(map[string]any)
	detail := agentGet(t, mux, base+"/nodes/"+failure["id"].(string))
	msg := detail["messages"].(map[string]any)["items"].([]any)[0].(map[string]any)
	refs := msg["screenshots"].([]any)
	if len(refs) != 1 || !strings.Contains(refs[0].(string), "screenshots%2Fexample.png") {
		t.Fatalf("screenshots: %v", refs)
	}
	if detail["start"] != "2026-09-10T10:00:00" {
		t.Fatalf("timestamp: %v", detail)
	}
	errors := agentGet(t, mux, base+"/errors")["errors"].(map[string]any)
	if errors["total"] != float64(1) {
		t.Fatalf("errors: %v", errors)
	}
	failed := agentGet(t, mux, base+"/tests?status=FAIL&offset=2&limit=1")["tests"].(map[string]any)
	if failed["total"] != float64(4) || failed["nextOffset"] != float64(3) {
		t.Fatalf("failed tests: %v", failed)
	}
}

func TestAgentEvidenceTextIsBoundedAndRecoverable(t *testing.T) {
	message := strings.Repeat("ä", 4500)
	xml := `<robot><suite name="S"><test name="T"><kw name="Fail"><arg>` + message + `</arg><msg level="FAIL">` + message + `</msg><msg level="INFO">second</msg><status status="FAIL">` + message + `</status></kw><status status="FAIL">` + message + `</status></test><status status="FAIL"/></suite></robot>`
	mux, id := agentTestServer(t, xml)
	path := "/api/agent/runs/" + id + "/nodes/s0-t0-k0?limit=1"
	compact := agentGet(t, mux, path)["messages"].(map[string]any)["items"].([]any)[0].(map[string]any)["text"].(map[string]any)
	if len([]rune(compact["text"].(string))) != 512 || compact["nextOffset"] != float64(512) {
		t.Fatalf("default compact preview: %v", compact)
	}
	var recovered strings.Builder
	for _, offset := range []string{"0", "2000", "4000"} {
		data := agentGet(t, mux, path+"&textOffset="+offset+"&textLimit=2000")
		m := data["messages"].(map[string]any)
		if m["nextOffset"] != float64(1) {
			t.Fatalf("message pagination: %v", m)
		}
		text := m["items"].([]any)[0].(map[string]any)["text"].(map[string]any)
		recovered.WriteString(text["text"].(string))
		if len([]rune(text["text"].(string))) > 2000 || text["totalChars"] != float64(4500) {
			t.Fatalf("text paging: %v", text)
		}
	}
	if recovered.String() != message {
		t.Fatal("text paging lost content")
	}
	second := agentGet(t, mux, path+"&offset=1")["messages"].(map[string]any)
	if second["nextOffset"] != nil {
		t.Fatalf("second page: %v", second)
	}
}

func TestAgentRoutesRejectInvalidRequests(t *testing.T) {
	mux, id := agentTestServer(t, agentFixture)
	base := "/api/agent/runs/" + id
	cases := []struct {
		method, path string
		code         int
	}{
		{"POST", base + "/triage", 405}, {"DELETE", base + "/tests/s0-t0", 405},
		{"GET", base + "/triage?limit=101", 400}, {"GET", base + "/triage?limit=0", 400},
		{"GET", base + "/triage?offset=-1", 400}, {"GET", base + "/triage?textOffset=no", 400},
		{"GET", base + "/tests?status=UNKNOWN", 400}, {"GET", base + "/nodes/missing", 404},
		{"GET", base + "/tests/missing", 404}, {"GET", base + "/groups/missing/tests", 404},
		{"GET", "/api/agent/runs/missing/triage", 404}, {"GET", base + "/delete", 404},
	}
	for _, c := range cases {
		t.Run(c.method+c.path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, httptest.NewRequest(c.method, c.path, nil))
			if rr.Code != c.code {
				t.Fatalf("got %d want %d: %s", rr.Code, c.code, rr.Body.String())
			}
		})
	}
}

func TestAgentIndexRefreshesForReparsedArtifact(t *testing.T) {
	first, err := rdiff.ParseRobotXMLBytes([]byte(agentFixture))
	if err != nil {
		t.Fatal(err)
	}
	second, err := rdiff.ParseRobotXMLBytes([]byte(`<robot><suite name="S"><test name="Fixed"><status status="PASS"/></test><status status="PASS"/></suite></robot>`))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer("", nil)
	original := server.agentIndexFor(first)
	if server.agentIndexFor(first) != original {
		t.Fatal("unchanged parsed run should reuse index")
	}
	refreshed := server.agentIndexFor(second)
	if refreshed == original || refreshed.counts["fail"] != 0 || refreshed.counts["pass"] != 1 {
		t.Fatal("stale index after replacing artifact")
	}
}
