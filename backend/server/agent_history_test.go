package backend

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	rdiff "robot_diff/backend/diff"
	"robot_diff/backend/store"
)

func historyRun(tests string) string {
	return `<robot><suite name="Root"><suite name="Login">` + tests + `<status status="FAIL"/></suite><status status="FAIL"/></suite></robot>`
}

func historyTest(name, status, msg string) string {
	return `<test name="` + name + `"><kw name="Step"><status status="` + status + `">` + msg + `</status></kw><status status="` + status + `">` + msg + `</status></test>`
}

// agentHistoryServer writes runs oldest first, one minute apart.
func agentHistoryServer(t *testing.T, runs ...string) (*http.ServeMux, *store.RunStore, []string) {
	t.Helper()
	root := t.TempDir()
	base := time.Now().Add(-time.Hour)
	paths := make([]string, len(runs))
	for i, xml := range runs {
		paths[i] = filepath.Join(root, "run"+string(rune('a'+i)), "output.xml")
		writeTestFile(t, paths[i], []byte(xml))
		when := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(paths[i], when, when); err != nil {
			t.Fatal(err)
		}
	}
	rs := store.NewRunStore(root, time.Hour)
	rs.ScanOnce()
	ids := make([]string, len(runs))
	for _, r := range rs.ListRuns() {
		for i, p := range paths {
			if filepath.Base(filepath.Dir(p)) == r.Name {
				ids[i] = r.ID
			}
		}
	}
	mux := http.NewServeMux()
	NewServer("", rs).registerRoutes(mux)
	return mux, rs, ids
}

func TestAgentCompareAndHistoryAcrossRuns(t *testing.T) {
	mux, _, ids := agentHistoryServer(t,
		historyRun(historyTest("A", "PASS", "")+historyTest("B", "FAIL", "boom")+historyTest("C", "PASS", "")+historyTest("D", "PASS", "")),
		historyRun(historyTest("A", "PASS", "")+historyTest("B", "FAIL", "boom")+historyTest("C", "FAIL", "x")+historyTest("D", "PASS", "")),
		historyRun(historyTest("A", "FAIL", "new problem")+historyTest("B", "FAIL", " boom ")+historyTest("C", "PASS", "")+historyTest("E", "FAIL", "added")),
	)
	current := "/api/agent/runs/" + ids[2]

	cmp := agentGet(t, mux, current+"/compare")
	if cmp["base"].(map[string]any)["id"] != ids[1] {
		t.Fatalf("default base should be the previous run: %v", cmp["base"])
	}
	counts := cmp["counts"].(map[string]any)
	want := map[string]float64{"newFailures": 2, "stillFailing": 1, "fixed": 1, "addedTests": 1, "removedTests": 1}
	for k, v := range want {
		if counts[k] != v {
			t.Fatalf("counts[%s] = %v, want %v (%v)", k, counts[k], v, counts)
		}
	}
	still := cmp["stillFailing"].(map[string]any)["items"].([]any)[0].(map[string]any)
	if still["sameFailure"] != true || still["test"].(map[string]any)["name"] != "Root.Login.B" {
		t.Fatalf("still failing: %v", still)
	}
	removed := cmp["removedTests"].(map[string]any)["items"].([]any)[0].(map[string]any)
	if removed["name"] != "Root.Login.D" || removed["baseStatus"] != "PASS" {
		t.Fatalf("removed: %v", removed)
	}

	explicit := agentGet(t, mux, current+"/compare?base="+ids[0])
	if c := explicit["counts"].(map[string]any); c["fixed"] != float64(0) || c["newFailures"] != float64(2) {
		t.Fatalf("compare against oldest run: %v", c)
	}

	// Test IDs are positional: s0 is Root, s0-s0 is Login.
	hist := agentGet(t, mux, current+"/tests/s0-s0-t0/history")
	summary := hist["summary"].(map[string]any)
	if summary["pattern"] != "new-failure" || summary["checked"] != float64(2) || summary["sameStatusStreak"] != float64(0) {
		t.Fatalf("history of new failure: %v", summary)
	}
	if summary["lastPass"].(map[string]any)["id"] != ids[1] {
		t.Fatalf("lastPass: %v", summary["lastPass"])
	}
	runs := hist["runs"].(map[string]any)["items"].([]any)
	if runs[0].(map[string]any)["run"].(map[string]any)["id"] != ids[1] || runs[0].(map[string]any)["testId"] != "s0-s0-t0" {
		t.Fatalf("history should be newest first with that run's test ID: %v", runs)
	}

	hist = agentGet(t, mux, current+"/tests/s0-s0-t1/history?limit=1")
	summary = hist["summary"].(map[string]any)
	if summary["pattern"] != "consistently-failing" || summary["sameStatusStreak"] != float64(1) {
		t.Fatalf("history of persistent failure: %v", summary)
	}
	page := hist["runs"].(map[string]any)
	if page["total"] != float64(2) || page["nextOffset"] != float64(1) {
		t.Fatalf("history paging: %v", page)
	}
	if runs := page["items"].([]any); runs[0].(map[string]any)["sameFailureAsCurrent"] != true {
		t.Fatalf("same failure: %v", runs)
	}

	pack := agentGet(t, mux, current+"/analysis-pack")
	comparison := pack["comparison"].(map[string]any)
	if comparison["counts"].(map[string]any)["newFailures"] != float64(2) {
		t.Fatalf("pack comparison: %v", comparison)
	}
	total := 0.0
	for _, g := range pack["groups"].([]any) {
		total += g.(map[string]any)["newFailureCount"].(float64)
	}
	if total != 2 {
		t.Fatalf("groups should mark the two new failures: %v", pack["groups"])
	}
}

func TestAgentCompareWithoutEarlierRun(t *testing.T) {
	mux, _, ids := agentHistoryServer(t, historyRun(historyTest("A", "FAIL", "x")))
	cmp := agentGet(t, mux, "/api/agent/runs/"+ids[0]+"/compare")
	if cmp["base"] != nil || cmp["note"] == nil {
		t.Fatalf("compare without history: %v", cmp)
	}
	hist := agentGet(t, mux, "/api/agent/runs/"+ids[0]+"/tests/s0-s0-t0/history")
	if hist["summary"].(map[string]any)["pattern"] != "no-history" {
		t.Fatalf("history without earlier runs: %v", hist["summary"])
	}
}

// The light reader must produce the same IDs and names as the agent index,
// including repeated test names, or history would follow the wrong test.
func TestTestOutcomesMatchAgentIndex(t *testing.T) {
	_, rs, ids := agentHistoryServer(t, agentFixture)
	got, err := rs.TestOutcomes(context.Background(), ids[0])
	if err != nil {
		t.Fatal(err)
	}
	robot, err := rdiff.ParseRobotXMLBytes([]byte(agentFixture))
	if err != nil {
		t.Fatal(err)
	}
	want := newAgentIndex(&robot.Suite).outcomes()
	if len(got) != len(want) {
		t.Fatalf("got %d outcomes, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].Name != want[i].Name || got[i].Status != want[i].Status || got[i].Message != want[i].Message {
			t.Fatalf("outcome %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
