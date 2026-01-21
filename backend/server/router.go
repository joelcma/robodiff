package backend

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/delete-runs", s.handleDeleteRuns)
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/test-details", s.handleTestDetails)
	mux.HandleFunc("/api/http-try", s.handleHTTPTry)
	mux.HandleFunc("/api/diff", s.handleDiff)
}