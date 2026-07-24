package post

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// cursorPayload is the wire form of a feed position. Keys are short because the
// encoded string travels in the URL on every page request.
type cursorPayload struct {
	P time.Time `json:"p"`
	I uuid.UUID `json:"i"`
}

func encodeCursor(c feedCursor) string {
	raw, err := json.Marshal(cursorPayload{P: c.publishedAt, I: c.id})
	if err != nil {
		// The payload is two fixed, always-marshalable fields, so this cannot
		// fail in practice; an empty cursor just ends the feed.
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// decodeCursor parses a cursor, returning nil for an empty one (the first page).
// A malformed cursor is a client error, not a server crash.
func decodeCursor(raw string) (*feedCursor, error) {
	if raw == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, httpx.New(httpx.CodeInvalidRequest, "Invalid cursor", err)
	}
	var payload cursorPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, httpx.New(httpx.CodeInvalidRequest, "Invalid cursor", err)
	}
	return &feedCursor{publishedAt: payload.P, id: payload.I}, nil
}

// parsePostID turns a path parameter into a UUID, with a clean 400 on garbage.
func parsePostID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.New(httpx.CodeInvalidRequest, "Invalid post id", fmt.Errorf("parse post id: %w", err))
	}
	return id, nil
}
