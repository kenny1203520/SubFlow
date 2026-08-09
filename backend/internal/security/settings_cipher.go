package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// SettingsCipher keeps provider credentials out of readable PocketBase
// records. The deployment key is deliberately never persisted by SubFlow.
type SettingsCipher struct{ key []byte }

func NewSettingsCipher(secret string) SettingsCipher {
	if secret == "" {
		return SettingsCipher{}
	}
	sum := sha256.Sum256([]byte(secret))
	return SettingsCipher{key: sum[:]}
}

func (c SettingsCipher) Available() bool { return len(c.key) == 32 }

func (c SettingsCipher) Encrypt(value string) (string, error) {
	if !c.Available() {
		return "", errors.New("settings encryption key is not configured")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(append(nonce, gcm.Seal(nil, nonce, []byte(value), nil)...)), nil
}

func (c SettingsCipher) Decrypt(value string) (string, error) {
	if !c.Available() {
		return "", errors.New("settings encryption key is not configured")
	}
	raw, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted settings value")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
