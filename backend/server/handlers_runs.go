package backend

import (
	"encoding/json"
	"net/http"
)

type deleteRunsRequest struct {
	RunIDs []string `json:"runIds"`
}

func (s *Server) handleDeleteRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req deleteRunsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.RunIDs) == 0 {
		writeError(w, http.StatusBadRequest, "runIds required")
		return
	}

	deleted, err := s.store.DeleteRuns(req.RunIDs)
	if err != nil {
		status, code, msg, detail := classifyError(err)
		writeErrorWithCode(w, status, code, msg, detail)
		return
	}

	// Refresh immediately so the UI sees the deletion on the next /api/runs.
	s.store.ScanOnce()

	writeJSON(w, http.StatusOK, map[string]any{
		"deleted": deleted,
	})
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	cfg := s.store.Config()
	runs := s.store.ListRuns()
	writeJSON(w, http.StatusOK, map[string]any{
		"dir":  cfg.Dir,
		"runs": runs,
	})
}