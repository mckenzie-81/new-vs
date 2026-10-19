package smtp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

const keySize = 32 // AES-256

// ErrCredentialKeyNotConfigured means SMTP_CREDENTIAL_KEY isn't set (or
// isn't a valid 32-byte base64 key) on this server - saving a password
// is refused rather than falling back to storing it in plain text, since
// a leaked SMTP credential lets someone send mail as the institution
// itself.
var ErrCredentialKeyNotConfigured = errors.New("the server has no SMTP_CREDENTIAL_KEY configured - set one (32 random bytes, base64-encoded) before saving an SMTP password")

// decodeKey parses SMTP_CREDENTIAL_KEY's value. An empty or malformed
// key isn't a startup error - most deployments won't have email set up
// at all yet - it just means Service.decryptKey is nil and any attempt
// to save a password fails clearly instead of silently going to
// plaintext.
func decodeKey(base64Key string) []byte {
	if base64Key == "" {
		return nil
	}
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil || len(key) != keySize {
		return nil
	}
	return key
}

// encrypt returns a base64 string: the random nonce followed by the
// AES-256-GCM-sealed ciphertext. A fresh random nonce every call means
// encrypting the same password twice never produces the same output,
// which is the property GCM needs to stay secure.
func encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func decrypt(key []byte, encoded string) (string, error) {
	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decoding ciphertext: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(sealed) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := sealed[:nonceSize], sealed[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting: %w", err)
	}
	return string(plaintext), nil
}
