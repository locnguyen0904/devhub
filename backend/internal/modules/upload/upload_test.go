package upload

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
	"github.com/locnguyen0904/devhub/backend/internal/platform/storage"
)

type fakePresigner struct{ called bool }

func (f *fakePresigner) PresignPut(_ context.Context, key, _ string, _ time.Duration) (storage.Presigned, error) {
	f.called = true
	return storage.Presigned{UploadURL: "http://s3/" + key, PublicURL: "http://cdn/" + key, ExpiresIn: 300}, nil
}

func TestPresignRejectsUnsupportedType(t *testing.T) {
	store := &fakePresigner{}
	svc := New(store)

	_, err := svc.Presign(context.Background(), uuid.New(), "application/pdf", 1000)

	if !isValidation(err) {
		t.Errorf("Presign(pdf) error = %v, want validation_failed", err)
	}
	if store.called {
		t.Error("Presign must reject the type before signing a URL")
	}
}

func TestPresignRejectsTooLarge(t *testing.T) {
	svc := New(&fakePresigner{})

	_, err := svc.Presign(context.Background(), uuid.New(), "image/png", 6*1024*1024)

	if !isValidation(err) {
		t.Errorf("Presign(6MB) error = %v, want validation_failed", err)
	}
}

func TestPresignRejectsZeroSize(t *testing.T) {
	svc := New(&fakePresigner{})

	_, err := svc.Presign(context.Background(), uuid.New(), "image/png", 0)

	if !isValidation(err) {
		t.Errorf("Presign(0 bytes) error = %v, want validation_failed", err)
	}
}

func TestPresignAcceptsValidImage(t *testing.T) {
	store := &fakePresigner{}
	svc := New(store)
	userID := uuid.New()

	got, err := svc.Presign(context.Background(), userID, "image/png", 1000)
	if err != nil {
		t.Fatalf("Presign() error = %v", err)
	}
	if !store.called {
		t.Error("Presign should sign a URL for a valid image")
	}
	// The key must be namespaced under the user, so uploads cannot collide or be
	// written into another user's prefix.
	wantPrefix := "http://s3/uploads/" + userID.String() + "/"
	if len(got.UploadURL) < len(wantPrefix) || got.UploadURL[:len(wantPrefix)] != wantPrefix {
		t.Errorf("UploadURL = %q, want prefix %q", got.UploadURL, wantPrefix)
	}
}

func isValidation(err error) bool {
	var h *httpx.Error
	return errors.As(err, &h) && h.Code == httpx.CodeValidationFailed
}
