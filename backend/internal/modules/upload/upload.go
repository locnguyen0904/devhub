// Package upload presigns image uploads so the browser can PUT directly to S3.
package upload

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/locnguyen0904/devhub/backend/internal/platform/random"
	"github.com/locnguyen0904/devhub/backend/internal/platform/storage"
)

const (
	presignTTL = 5 * time.Minute
	maxBytes   = 5 * 1024 * 1024 // 5 MB
)

// extensionFor maps an accepted image content type to its file extension.
// Anything not listed is rejected before a URL is signed, so only images upload.
func extensionFor(contentType string) (string, bool) {
	switch contentType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/webp":
		return ".webp", true
	case "image/gif":
		return ".gif", true
	default:
		return "", false
	}
}

type presigner interface {
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (storage.Presigned, error)
}

// Service presigns validated image uploads.
type Service struct {
	store presigner
}

// New builds the service.
func New(store presigner) *Service {
	return &Service{store: store}
}

// Presign validates the request and returns a presigned PUT for one object. The
// key is namespaced by user and randomised so uploads never collide and cannot
// be guessed.
func (s *Service) Presign(ctx context.Context, userID uuid.UUID, contentType string, sizeBytes int64) (storage.Presigned, error) {
	ext, ok := extensionFor(contentType)
	if !ok {
		return storage.Presigned{}, errUnsupportedType
	}
	if sizeBytes <= 0 || sizeBytes > maxBytes {
		return storage.Presigned{}, errTooLarge
	}

	name, err := random.Hex(16)
	if err != nil {
		return storage.Presigned{}, err
	}
	key := path.Join("uploads", userID.String(), name+ext)

	presigned, err := s.store.PresignPut(ctx, key, contentType, presignTTL)
	if err != nil {
		return storage.Presigned{}, fmt.Errorf("presign upload: %w", err)
	}
	return presigned, nil
}
