// Package token issues and verifies the short-lived access JWTs.
//
// Refresh tokens are deliberately not JWTs — they are opaque random strings the
// auth module stores hashed, so they can be revoked. A JWT cannot be revoked
// before it expires, which is fine for a 15-minute access token but wrong for a
// 30-day refresh token.
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalid is returned for any token that fails to verify: bad signature,
// expired, malformed, or wrong signing method.
var ErrInvalid = errors.New("invalid token")

// errWrongSigningMethod guards against algorithm-substitution attacks.
var errWrongSigningMethod = errors.New("unexpected signing method")

// Claims is the access-token payload. jti lets a specific token be traced in
// logs; it is not a revocation list, since access tokens live only minutes.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// UserID returns the authenticated user id carried in the subject claim.
func (c Claims) UserID() (uuid.UUID, error) {
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse subject as uuid: %w", err)
	}
	return id, nil
}

// Issuer signs and verifies access tokens with a single HMAC secret.
type Issuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewIssuer builds an Issuer. now is injectable so tests can control expiry.
func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl, now: time.Now}
}

// Issue signs an access token for the user. It returns the signed string and
// the moment it expires, so the handler can tell the client expires_in.
func (i *Issuer) Issue(userID uuid.UUID, username string) (string, time.Time, error) {
	now := i.now()
	expiresAt := now.Add(i.ttl)

	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

// Verify parses and validates a token, returning its claims. Every failure
// collapses to ErrInvalid so callers cannot leak which check failed.
func (i *Issuer) Verify(raw string) (Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		// Reject any algorithm other than the one we sign with. Without this
		// check an attacker could submit a token signed with "none" or swap
		// HMAC for RSA and bypass verification.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", errWrongSigningMethod, t.Header["alg"])
		}
		return i.secret, nil
	}, jwt.WithTimeFunc(i.now))
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	return claims, nil
}
