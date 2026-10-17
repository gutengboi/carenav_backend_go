package cryptoutil

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Hash one-way hashes a password or PIN with bcrypt.
func Hash(secret string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	return string(b), err
}

// Verify checks secret against a stored value that is either a bcrypt hash
// (current format, prefixed "$2") or legacy AES ciphertext from before the
// hashing migration. needsRehash reports whether the match came from the
// legacy path, so the caller can upgrade the stored value.
func Verify(stored, secret, legacySecret string) (ok, needsRehash bool) {
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(secret)) == nil, false
	}
	decrypted, err := Decrypt(stored, legacySecret)
	if err != nil {
		return false, false
	}
	return decrypted == secret, true
}
