package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidKey is returned when the encryption key is invalid
	ErrInvalidKey = errors.New("invalid encryption key")
	// ErrInvalidCiphertext is returned when the ciphertext is invalid
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrDecryption is returned when decryption fails
	ErrDecryption = errors.New("decryption failed")
)

// Encryptor defines the interface for encryption/decryption operations
type Encryptor interface {
	// Encrypt encrypts plaintext and returns base64 encoded ciphertext
	Encrypt(plaintext string) (string, error)
	// Decrypt decrypts base64 encoded ciphertext and returns plaintext
	Decrypt(ciphertext string) (string, error)
}

// AESGCMEncryptor implements Encryptor using AES-256-GCM
// GCM (Galois/Counter Mode) provides authenticated encryption
type AESGCMEncryptor struct {
	key []byte
}

// NewAESGCMEncryptor creates a new AES-256-GCM encryptor
// Key must be 32 bytes (256 bits) for AES-256
func NewAESGCMEncryptor(key []byte) (*AESGCMEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: key must be 32 bytes for AES-256", ErrInvalidKey)
	}

	return &AESGCMEncryptor{
		key: key,
	}, nil
}

// NewAESGCMEncryptorFromString creates a new AES-256-GCM encryptor from a base64 encoded key
func NewAESGCMEncryptorFromString(encodedKey string) (*AESGCMEncryptor, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode key: %v", ErrInvalidKey, err)
	}

	return NewAESGCMEncryptor(key)
}

// Encrypt encrypts plaintext using AES-256-GCM
// Returns base64 encoded string: base64(nonce + ciphertext + tag)
func (e *AESGCMEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("plaintext cannot be empty")
	}

	// Create AES cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	// The nonce size must be exactly gcm.NonceSize() bytes
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	// GCM automatically appends the authentication tag to the ciphertext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for storage
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return encoded, nil
}

// Decrypt decrypts base64 encoded ciphertext using AES-256-GCM
// Expects format: base64(nonce + ciphertext + tag)
func (e *AESGCMEncryptor) Decrypt(encodedCiphertext string) (string, error) {
	if encodedCiphertext == "" {
		return "", errors.New("ciphertext cannot be empty")
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", fmt.Errorf("%w: failed to decode ciphertext: %v", ErrInvalidCiphertext, err)
	}

	// Create AES cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Verify minimum size: nonce + at least 1 byte + tag
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryption, err)
	}

	return string(plaintext), nil
}

// GenerateKey generates a cryptographically secure random 256-bit key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // 256 bits
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyString generates a base64 encoded random 256-bit key
func GenerateKeyString() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// RotatableEncryptor supports key rotation for encrypted data
type RotatableEncryptor struct {
	current  *AESGCMEncryptor
	previous *AESGCMEncryptor // For decrypting old data during rotation
}

// NewRotatableEncryptor creates a new encryptor with key rotation support
func NewRotatableEncryptor(currentKey, previousKey []byte) (*RotatableEncryptor, error) {
	current, err := NewAESGCMEncryptor(currentKey)
	if err != nil {
		return nil, fmt.Errorf("invalid current key: %w", err)
	}

	var previous *AESGCMEncryptor
	if previousKey != nil {
		previous, err = NewAESGCMEncryptor(previousKey)
		if err != nil {
			return nil, fmt.Errorf("invalid previous key: %w", err)
		}
	}

	return &RotatableEncryptor{
		current:  current,
		previous: previous,
	}, nil
}

// Encrypt always uses the current key
func (r *RotatableEncryptor) Encrypt(plaintext string) (string, error) {
	return r.current.Encrypt(plaintext)
}

// Decrypt tries the current key first, then falls back to the previous key
func (r *RotatableEncryptor) Decrypt(ciphertext string) (string, error) {
	// Try current key first
	plaintext, err := r.current.Decrypt(ciphertext)
	if err == nil {
		return plaintext, nil
	}

	// If there's no previous key, return the error
	if r.previous == nil {
		return "", err
	}

	// Try previous key for backwards compatibility during rotation
	plaintext, err = r.previous.Decrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decryption failed with both current and previous keys")
	}

	return plaintext, nil
}

// ReEncrypt decrypts with old key and encrypts with new key (for key rotation)
func (r *RotatableEncryptor) ReEncrypt(oldCiphertext string) (string, error) {
	// Decrypt with either key
	plaintext, err := r.Decrypt(oldCiphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	// Re-encrypt with current key
	newCiphertext, err := r.current.Encrypt(plaintext)
	if err != nil {
		return "", fmt.Errorf("failed to re-encrypt: %w", err)
	}

	return newCiphertext, nil
}
