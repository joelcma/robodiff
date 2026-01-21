package backend

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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

	httpReq, err := http.NewRequestWithContext(r.Context(), method, u.String(), bytes.NewReader([]byte(req.Body)))
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

	client := &http.Client{Timeout: 0}
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