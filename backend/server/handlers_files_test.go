package backend

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"robot_diff/backend/store"
)

func TestHandleRunFileFindsPabotWorkerScreenshot(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "smoke_20260903_130156_pabot")
	writeTestFile(t, filepath.Join(runDir, "output.xml"), []byte(
		`<robot><suite name="Suite"><status status="PASS"/></suite></robot>`,
	))
	screenshotName := "screenshot-example.png"
	screenshot := []byte("pabot screenshot")
	writeTestFile(t, filepath.Join(
		runDir, "pabot_results", "1", "screenshots", screenshotName,
	), screenshot)

	runStore := store.NewRunStore(root, time.Hour)
	runStore.ScanOnce()
	runs := runStore.ListRuns()
	if len(runs) != 1 {
		t.Fatalf("ListRuns() returned %d runs, want 1", len(runs))
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/run-file?runId="+url.QueryEscape(runs[0].ID)+
			"&path="+url.QueryEscape("screenshots/"+screenshotName),
		nil,
	)
	response := httptest.NewRecorder()
	NewServer("", runStore).handleRunFile(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Body.Bytes(); string(got) != string(screenshot) {
		t.Fatalf("body = %q, want %q", got, screenshot)
	}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
