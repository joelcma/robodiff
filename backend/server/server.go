package backend

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	rdiff "robot_diff/backend/diff"
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
	mux.HandleFunc("/api/test-details", s.handleTestDetails)
	mux.HandleFunc("/api/http-try", s.handleHTTPTry)
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

type testDetailsRequest struct {
	RunID    string `json:"runId"`
	TestName string `json:"testName"`
}

type diffRequest struct {
	RunIDs []string `json:"runIds"`
	Title  string   `json:"title"`
}

type httpTryRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type httpTryResponse struct {
	Status     int                 `json:"status"`
	StatusText string              `json:"statusText"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
	DurationMs int64               `json:"durationMs"`
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
		"keywords": buildKeywordsData(test.Keywords),
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleHTTPTry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req httpTryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	if !isAllowedHTTPMethod(method) {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}

	urlStr := strings.TrimSpace(req.URL)
	if urlStr == "" {
		writeError(w, http.StatusBadRequest, "url required")
		return
	}
	u, err := url.Parse(urlStr)
	if err != nil || u.Scheme == "" || u.Host == "" {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		writeError(w, http.StatusBadRequest, "only http/https supported")
		return
	}

	const maxBodyBytes = 1024 * 1024 // 1MB
	if len(req.Body) > maxBodyBytes {
		writeError(w, http.StatusBadRequest, "request body too large")
		return
	}

	httpReq, err := http.NewRequest(method, u.String(), bytes.NewReader([]byte(req.Body)))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to build request")
		return
	}
	for k, v := range req.Headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		kl := strings.ToLower(strings.TrimSpace(k))
		// Let Go manage gzip transparently; otherwise we risk returning compressed bytes.
		if kl == "accept-encoding" {
			continue
		}
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()

	headers := resp.Header.Clone()
	bodyBytes, _ := readDecodedResponseBody(resp, maxBodyBytes)
	if strings.TrimSpace(headers.Get("Content-Encoding")) != "" {
		// If we decoded, remove encoding headers so headers/body match.
		headers.Del("Content-Encoding")
		headers.Del("Content-Length")
	}
	data := map[string]any{
		"request": map[string]any{
			"method":  method,
			"url":     u.String(),
			"headers": req.Headers,
			"body":    req.Body,
		},
		"response": httpTryResponse{
			Status:     resp.StatusCode,
			StatusText: resp.Status,
			Headers:    headers,
			Body:       string(bodyBytes),
			DurationMs: duration.Milliseconds(),
		},
	}
	writeJSON(w, http.StatusOK, data)
}

func readDecodedResponseBody(resp *http.Response, limit int64) ([]byte, error) {
	enc := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	// If enc is empty, resp.Body is already plain.
	if enc == "" {
		return io.ReadAll(io.LimitReader(resp.Body, limit))
	}

	switch enc {
	case "gzip":
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return io.ReadAll(io.LimitReader(resp.Body, limit))
		}
		defer gr.Close()
		return io.ReadAll(io.LimitReader(gr, limit))
	case "deflate":
		zr, err := zlib.NewReader(resp.Body)
		if err != nil {
			return io.ReadAll(io.LimitReader(resp.Body, limit))
		}
		defer zr.Close()
		return io.ReadAll(io.LimitReader(zr, limit))
	default:
		// Unknown encoding; return raw bytes to avoid corrupting data.
		return io.ReadAll(io.LimitReader(resp.Body, limit))
	}
}

func isAllowedHTTPMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func buildKeywordsData(keywords []rdiff.Keyword) []map[string]any {
	result := make([]map[string]any, len(keywords))
	for i, kw := range keywords {
		result[i] = map[string]any{
			"name":      kw.Name,
			"type":      kw.Type,
			"status":    kw.Status.Status,
			"start":     kw.Status.StartTime,
			"end":       kw.Status.EndTime,
			"arguments": kw.Arguments,
			"keywords":  buildKeywordsData(kw.Keywords),
			"messages":  buildMessagesData(kw.Messages),
		}
	}
	return result
}

func buildMessagesData(messages []rdiff.Message) []map[string]any {
	result := make([]map[string]any, len(messages))
	for i, msg := range messages {
		result[i] = map[string]any{
			"level":     msg.Level,
			"timestamp": msg.Timestamp,
			"text":      msg.Text,
		}
	}
	return result
}

func buildSuitesData(suite *rdiff.Suite) []map[string]any {
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

	results := rdiff.NewDiffResults()
	for i := range robots {
		results.AddParsedOutput(robots[i], columns[i])
	}

	reporter := rdiff.NewDiffReporter("", req.Title, columns, inputFiles, rdiff.Templates{})
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
