package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const robotOutput = `<robot><suite name="Suite"><status status="PASS"/></suite></robot>`

func TestScanOnceIgnoresPabotWorkerOutputs(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "smoke_20260903_130156_pabot")
	writeRobotOutput(t, filepath.Join(runDir, "output.xml"))
	writeRobotOutput(t, filepath.Join(runDir, "pabot_results", "0", "output.xml"))
	writeRobotOutput(t, filepath.Join(runDir, "pabot_results", "1", "output.xml"))

	store := newTestRunStore(root)
	store.ScanOnce()

	runs := store.ListRuns()
	if len(runs) != 1 {
		t.Fatalf("ListRuns() returned %d runs, want only the merged Pabot run", len(runs))
	}
	if got, want := runs[0].Name, "smoke_20260903_130156_pabot"; got != want {
		t.Fatalf("run name = %q, want %q", got, want)
	}
	if got, want := runs[0].RelPath, "smoke_20260903_130156_pabot/output.xml"; got != want {
		t.Fatalf("run path = %q, want %q", got, want)
	}
}

func TestScanOnceDoesNotExposeIncompletePabotWorkers(t *testing.T) {
	root := t.TempDir()
	workerDir := filepath.Join(root, "smoke_20260903_130156_pabot", "pabot_results")
	writeRobotOutput(t, filepath.Join(workerDir, "0", "output.xml"))
	writeRobotOutput(t, filepath.Join(workerDir, "1", "output.xml"))

	store := newTestRunStore(root)
	store.ScanOnce()

	if runs := store.ListRuns(); len(runs) != 0 {
		t.Fatalf("ListRuns() returned %d worker artifacts before merge, want 0", len(runs))
	}
}

func newTestRunStore(root string) *RunStore {
	return &RunStore{
		dir:      root,
		interval: time.Hour,
		runs:     make(map[string]*runEntry),
	}
}

func writeRobotOutput(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(robotOutput), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
