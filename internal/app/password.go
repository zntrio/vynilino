package app

import (
	"github.com/alexedwards/argon2id"

	"zntr.io/vynilino/internal/domain"
)

var argon2Params = &argon2id.Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// hashPassword returns an Argon2id hash of the plaintext password.
func hashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2Params)
}

// checkPassword verifies a plaintext password against an Argon2id hash.
// Returns domain.ErrInvalidCredentials on mismatch.
func checkPassword(password, hash string) error {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return domain.ErrInvalidCredentials
	}
	if !match {
		return domain.ErrInvalidCredentials
	}
	return nil
}

// dummyHash is used for constant-time rejection when a user is not found.
const dummyHash = "$argon2id$v=19$m=65536,t=3,p=2$c29tZXNhbHRzb21lc2FsdA$" +
	"Nf1LMRkJP5aJOhkxQdCYRSNxfYGgQCgBbnVi6KkiNY"

// constantTimeReject performs a dummy hash comparison to prevent timing attacks.
func constantTimeReject() {
	_, _ = argon2id.ComparePasswordAndHash("dummy_password_for_timing", dummyHash)
}

// HashPassword hashes a plaintext password using the application's Argon2id parameters.
// Exported for use in admin CLI tools.
func HashPassword(password string) (string, error) {
	return hashPassword(password)
}

// ValidatePasswordStrength validates that a password meets complexity requirements.
// Exported for use in admin CLI tools.
func ValidatePasswordStrength(password string) error {
	return validatePassword(password)
}
