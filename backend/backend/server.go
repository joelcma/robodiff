package backend

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"robot_diff/backend/robotdiff"
)

type Server struct {
	store *RunStore
	addr  string
}

func NewServer(addr string, store *RunStore) *Server {
	return &Server{addr: addr, store: store}
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()

	// API
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/diff", s.handleDiff)

	// UI/static
	ui, err := newUIHandler()
	if err != nil {
		return err
	}
	mux.Handle("/", ui)

	h := withCORS(mux)
	server := &http.Server{
		Addr:              s.addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"dir":          s.store.Dir(),
		"scanInterval": s.store.Interval().String(),
	})
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	runs := s.store.ListRuns()
	writeJSON(w, http.StatusOK, map[string]any{
		"dir":  s.store.Dir(),
		"runs": runs,
	})
}

type runRequest struct {
	RunID string `json:"runId"`
}

type diffRequest struct {
	RunIDs []string `json:"runIds"`
	Title  string   `json:"title"`
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

func buildSuitesData(suite *robotdiff.Suite) []map[string]any {
	var result []map[string]any
	
	// Add current suite if it has tests
	if len(suite.Tests) > 0 {
		tests := make([]map[string]any, len(suite.Tests))
		for i, test := range suite.Tests {
			tests[i] = map[string]any{
				"name":   test.Name,
				"status": test.Status.Status,
			}
		}
		result = append(result, map[string]any{
			"name":  suite.Name,
			"tests": tests,
		})
	}
	
	// Recursively add sub-suites
	for i := range suite.Suites {
		result = append(result, buildSuitesData(&suite.Suites[i])...)
	}
	
	return result
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
		req.Title = "Robot Diff"
	}

	columns, inputFiles, robots, err := s.store.GetRuns(req.RunIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	results := robotdiff.NewDiffResults()
	for i := range robots {
		results.AddParsedOutput(robots[i], columns[i])
	}

	reporter := robotdiff.NewDiffReporter("", req.Title, columns, inputFiles, robotdiff.Templates{})
	writeJSON(w, http.StatusOK, reporter.BuildJSONData(results))
}

func newUIHandler() (http.Handler, error) {
	uiFS, ok := uiFileSystem()
	if !ok {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintln(w, "UI not available. For dev: cd web && npm run dev. For prod: cd web && npm run build")
		}), nil
	}

	sub, err := fs.Sub(uiFS, ".")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		rw := &statusCapturingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		fileServer.ServeHTTP(rw, r)
		if rw.status != http.StatusNotFound {
			return
		}

		r2 := *r
		u := *r.URL
		u.Path = "/"
		r2.URL = &u
		fileServer.ServeHTTP(w, &r2)
	}), nil
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
