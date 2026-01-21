package backend

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

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