package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	// ErrInvalidHash is returned when the password hash format is invalid
	ErrInvalidHash = errors.New("invalid password hash format")
	// ErrMismatchedHashAndPassword is returned when password doesn't match hash
	ErrMismatchedHashAndPassword = errors.New("password does not match hash")
)

// Argon2Params defines the parameters for Argon2id hashing
// These parameters follow OWASP recommendations for password storage
type Argon2Params struct {
	Memory      uint32 // Memory in KiB (64 MB = 65536 KiB)
	Iterations  uint32 // Number of iterations
	Parallelism uint8  // Degree of parallelism
	SaltLength  uint32 // Length of random salt in bytes
	KeyLength   uint32 // Length of generated key in bytes
}

// DefaultArgon2Params returns the recommended parameters for Argon2id
// Memory: 64 MB, Iterations: 3, Parallelism: 2, Salt: 16 bytes, Key: 32 bytes
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// PasswordHasher defines the interface for password hashing operations
type PasswordHasher interface {
	// Hash creates a secure hash from the given password
	Hash(password string) (string, error)
	// Verify checks if the password matches the hash
	Verify(password, hash string) error
}

// Argon2Hasher implements PasswordHasher using Argon2id
type Argon2Hasher struct {
	params *Argon2Params
}

// NewArgon2Hasher creates a new Argon2Hasher with the given parameters
func NewArgon2Hasher(params *Argon2Params) *Argon2Hasher {
	if params == nil {
		params = DefaultArgon2Params()
	}
	return &Argon2Hasher{
		params: params,
	}
}

// Hash generates a secure Argon2id hash from the password
// Returns a hash in the format: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
func (h *Argon2Hasher) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// Generate a cryptographically secure random salt
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate the hash using Argon2id
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.params.Iterations,
		h.params.Memory,
		h.params.Parallelism,
		h.params.KeyLength,
	)

	// Encode salt and hash to base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.params.Memory,
		h.params.Iterations,
		h.params.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

// Verify checks if the given password matches the hash
// Uses constant-time comparison to prevent timing attacks
func (h *Argon2Hasher) Verify(password, encodedHash string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}
	if encodedHash == "" {
		return errors.New("hash cannot be empty")
	}

	// Parse the encoded hash to extract parameters and values
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return err
	}

	// Generate hash from the provided password using extracted parameters
	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return nil
	}

	return ErrMismatchedHashAndPassword
}

// decodeHash parses the encoded hash string and extracts parameters and values
func decodeHash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	// Verify algorithm
	if parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("unsupported algorithm: %s", parts[1])
	}

	// Parse version
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("incompatible version: %d", version)
	}

	// Parse parameters
	params := &Argon2Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Decode salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid salt encoding: %w", err)
	}
	params.SaltLength = uint32(len(salt))

	// Decode hash
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid hash encoding: %w", err)
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}

// ValidatePasswordStrength checks if password meets minimum security requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if len(password) > 128 {
		return errors.New("password must not exceed 128 characters")
	}

	// Check for at least one uppercase, one lowercase, one digit
	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}

	return nil
}
