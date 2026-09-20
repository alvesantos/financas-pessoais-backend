package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger registra método, rota, status e duração de cada requisição.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.InfoContext(r.Context(), "requisição",
			"metodo", r.Method,
			"rota", r.URL.Path,
			"status", rec.status,
			"duracao", time.Since(start).Round(time.Millisecond).String(),
		)
	})
}

// statusRecorder guarda o status escrito para o log.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.written {
		return
	}
	r.status = code
	r.written = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.written = true
	return r.ResponseWriter.Write(b)
}
