package backend

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) handleRunFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	runID := strings.TrimSpace(r.URL.Query().Get("runId"))
	relPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if runID == "" || relPath == "" {
		writeError(w, http.StatusBadRequest, "runId and path required")
		return
	}

	runFile, err := s.store.RunFilePath(runID)
	if err != nil {
		status, code, msg, detail := classifyError(err)
		writeErrorWithCode(w, status, code, msg, detail)
		return
	}

	clean := filepath.Clean(filepath.FromSlash(relPath))
	if clean == "." || clean == string(filepath.Separator) || filepath.IsAbs(clean) {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	if strings.Contains(clean, "..") {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	if !(clean == "screenshots" || strings.HasPrefix(clean, "screenshots"+string(filepath.Separator))) {
		writeError(w, http.StatusBadRequest, "only screenshots path allowed")
		return
	}

	baseDir := filepath.Dir(runFile)
	abs := filepath.Join(baseDir, clean)
	absClean, err := filepath.Abs(abs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid base dir")
		return
	}

	if !strings.HasPrefix(absClean, baseAbs+string(filepath.Separator)) && absClean != baseAbs {
		writeError(w, http.StatusBadRequest, "path escapes base")
		return
	}

	resolved, ok := resolveRunScreenshot(baseAbs, clean)
	if !ok {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	http.ServeFile(w, r, resolved)
}

func resolveRunScreenshot(baseDir, cleanPath string) (string, bool) {
	direct := filepath.Join(baseDir, cleanPath)
	if isRegularFile(direct) {
		return direct, true
	}

	// The merged Pabot output keeps screenshot references as screenshots/..., but
	// the files remain under pabot_results/<worker>/screenshots.
	workers, err := os.ReadDir(filepath.Join(baseDir, "pabot_results"))
	if err != nil {
		return "", false
	}
	for _, worker := range workers {
		if !worker.IsDir() {
			continue
		}
		candidate := filepath.Join(baseDir, "pabot_results", worker.Name(), cleanPath)
		if isRegularFile(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
