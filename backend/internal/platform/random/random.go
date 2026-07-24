// Package random generates unguessable strings from crypto/rand.
//
// Everything here must resist guessing: OAuth state, refresh tokens, username
// suffixes. math/rand is predictable and must never be used for these — a
// guessable OAuth state or refresh token means account takeover.
package random

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Token returns a URL-safe random string carrying nBytes of entropy. The
// base64url alphabet (A-Za-z0-9-_) is a strict subset of what usernames allow,
// so the output is also safe as a username suffix.
func Token(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
