// Package logger builds the application's structured logger.
package logger

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyUserID
)

// New returns a JSON logger in production and a human-readable one elsewhere.
// Both wrap the handler so request-scoped attributes are attached automatically.
func New(production bool) *slog.Logger {
	level := slog.LevelDebug
	if production {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}

	var h slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if production {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(&contextHandler{next: h})
}

// WithRequestID returns a context carrying the request id, which contextHandler
// then attaches to every log record made with that context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

// WithUserID returns a context carrying the authenticated user id.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, id)
}

// contextHandler copies request-scoped values onto each record so call sites
// never have to thread a logger through every function signature.
type contextHandler struct {
	next slog.Handler
}

func (h *contextHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.next.Enabled(ctx, l)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		r.AddAttrs(slog.String("request_id", v))
	}
	if v, ok := ctx.Value(ctxKeyUserID).(string); ok {
		r.AddAttrs(slog.String("user_id", v))
	}
	return h.next.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{next: h.next.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{next: h.next.WithGroup(name)}
}
