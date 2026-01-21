package backend

import (
	"encoding/json"
	"net/http"
)

type runRequest struct {
	RunID string `json:"runId"`
}

type testDetailsRequest struct {
	RunID    string `json:"runId"`
	TestName string `json:"testName"`
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.RunID == "" {
		writeError(w, http.StatusBadRequest, "runId required")
		return
	}

	columns, inputFiles, robots, err := s.store.GetRuns([]string{req.RunID})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	robot := robots[0]
	data := map[string]any{
		"title":  columns[0],
		"file":   inputFiles[0],
		"suites": buildSuitesData(&robot.Suite),
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleTestDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req testDetailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.RunID == "" || req.TestName == "" {
		writeError(w, http.StatusBadRequest, "runId and testName required")
		return
	}

	test, err := s.store.GetTestDetails(req.RunID, req.TestName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	data := map[string]any{
		"name":     test.Name,
		"status":   test.Status.Status,
		"start":    test.Status.StartTime,
		"end":      test.Status.EndTime,
		"keywords": buildKeywordsData(buildTestBodyKeywords(test)),
	}
	writeJSON(w, http.StatusOK, data)
}