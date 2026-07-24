package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/locnguyen0904/devhub/backend/internal/platform/logger"
)

// requestLogger logs one line per request and puts the request id into the
// context so every log made while handling that request carries it.
//
// This is the single place requests are logged. Lower layers return errors
// upward instead of logging them, otherwise one failure appears four times.
func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// chi's RequestID only puts the id in the context. Echoing it back is
			// what lets a user quote an id from a failed response so it can be
			// found in the logs — error bodies cannot carry it (see httpx).
			id := middleware.GetReqID(r.Context())
			ww.Header().Set("X-Request-ID", id)

			ctx := logger.WithRequestID(r.Context(), id)
			next.ServeHTTP(ww, r.WithContext(ctx))

			log.InfoContext(ctx, "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}
