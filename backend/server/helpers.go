package backend

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
)

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

func writeErrorWithCode(w http.ResponseWriter, status int, code, msg, detail string) {
	payload := map[string]string{
		"error": msg,
		"code":  code,
	}
	if detail != "" && detail != msg {
		payload["detail"] = detail
	}
	writeJSON(w, status, payload)
}

func classifyError(err error) (status int, code string, message string, detail string) {
	if err == nil {
		return http.StatusInternalServerError, "UNKNOWN", "Unknown error", ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return http.StatusRequestTimeout, "TIMEOUT", "Operation timed out", err.Error()
	}
	if errors.Is(err, os.ErrNotExist) {
		return http.StatusNotFound, "MISSING_FILE", "Run file no longer exists", err.Error()
	}

	msg := err.Error()
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "parse run") || strings.Contains(lower, "invalid xml") {
		return http.StatusUnprocessableEntity, "PARSE_ERROR", "Failed to parse Robot XML", msg
	}
	if strings.Contains(lower, "run not found") || strings.Contains(lower, "test not found") {
		return http.StatusNotFound, "NOT_FOUND", "Requested run or test not found", msg
	}

	return http.StatusBadRequest, "BAD_REQUEST", msg, ""
}