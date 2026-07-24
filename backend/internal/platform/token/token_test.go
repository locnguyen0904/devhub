package token

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

const testSecret = "test-secret-at-least-32-bytes-long-xx"

func TestIssueThenVerifyRoundTrips(t *testing.T) {
	iss := NewIssuer(testSecret, 15*time.Minute)
	userID := uuid.New()

	raw, expiresAt, err := iss.Issue(userID, "locnguyen")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Errorf("Issue() expiresAt = %v, want a future time", expiresAt)
	}

	claims, err := iss.Verify(raw)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	gotID, err := claims.UserID()
	if err != nil {
		t.Fatalf("UserID() error = %v", err)
	}
	if gotID != userID {
		t.Errorf("Verify() subject = %v, want %v", gotID, userID)
	}
	if claims.Username != "locnguyen" {
		t.Errorf("Verify() username = %q, want %q", claims.Username, "locnguyen")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	iss := &Issuer{secret: []byte(testSecret), ttl: time.Minute, now: func() time.Time { return past }}

	raw, _, err := iss.Issue(uuid.New(), "locnguyen")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	// Verify with the real clock: the token expired an hour before "now".
	live := NewIssuer(testSecret, time.Minute)
	if _, err := live.Verify(raw); !errors.Is(err, ErrInvalid) {
		t.Errorf("Verify(expired) error = %v, want ErrInvalid", err)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	raw, _, err := NewIssuer(testSecret, time.Minute).Issue(uuid.New(), "locnguyen")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	other := NewIssuer("a-completely-different-secret-value-xx", time.Minute)
	if _, err := other.Verify(raw); !errors.Is(err, ErrInvalid) {
		t.Errorf("Verify(wrong secret) error = %v, want ErrInvalid", err)
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	iss := NewIssuer(testSecret, time.Minute)
	if _, err := iss.Verify("not.a.jwt"); !errors.Is(err, ErrInvalid) {
		t.Errorf("Verify(garbage) error = %v, want ErrInvalid", err)
	}
}
