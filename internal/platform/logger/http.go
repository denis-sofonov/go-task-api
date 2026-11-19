package logger

import (
	"log/slog"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (w *responseRecorder) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(status)
	if w.status == 0 && status >= 200 {
		w.status = status
	}
}

func (w *responseRecorder) Write(body []byte) (int, error) {
	n, err := w.ResponseWriter.Write(body)
	if w.status == 0 {
		w.status = http.StatusOK
	}

	return n, err
}

func AccessLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		log.InfoContext(
			r.Context(),
			"http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"route", r.Pattern,
			"duration_ms", float64(time.Since(start))/float64(time.Millisecond))
	})
}
