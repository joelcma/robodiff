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

	info, err := os.Stat(absClean)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	http.ServeFile(w, r, absClean)
}
