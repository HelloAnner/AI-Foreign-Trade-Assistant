package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if skipLogging(r) {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		writer := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(writer, r)

		status := writer.Status()
		if status == 0 {
			status = http.StatusOK
		}
		duration := time.Since(start)

		if status >= 400 || r.Method != http.MethodGet || duration > 2*time.Second {
			log.Printf("[http] %s %s status=%d duration=%s", r.Method, r.URL.RequestURI(), status, duration.Round(50*time.Millisecond))
		}
	})
}

func skipLogging(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	path := r.URL.Path
	if path == "/" || path == "/index.html" {
		return true
	}
	if strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/favicon") || strings.HasPrefix(path, "/vite.svg") {
		return true
	}
	if strings.HasPrefix(path, "/api/") {
		return true
	}
	return false
}
