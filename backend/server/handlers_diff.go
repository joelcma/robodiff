package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	rdiff "robot_diff/backend/diff"
)

type diffRequest struct {
	RunIDs []string `json:"runIds"`
	Title  string   `json:"title"`
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req diffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.RunIDs) < 2 {
		writeError(w, http.StatusBadRequest, "select at least 2 runs")
		return
	}
	if req.Title == "" {
		req.Title = "Robodiff"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	columns, inputFiles, robots, err := s.store.GetRuns(ctx, req.RunIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	results := rdiff.NewDiffResults()
	for i := range robots {
		if err := ctx.Err(); err != nil {
			writeError(w, http.StatusRequestTimeout, err.Error())
			return
		}
		results.AddParsedOutput(robots[i], columns[i])
	}

	reporter := rdiff.NewDiffReporter(req.Title, columns, inputFiles)
	writeJSON(w, http.StatusOK, reporter.BuildJSONData(results))
}