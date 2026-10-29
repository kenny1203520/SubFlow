package security

import (
	"bytes"
	"testing"
)

func TestDeriveKeyringRequiresRootKey(t *testing.T) {
	if _, err := DeriveKeyring(""); err == nil {
		t.Fatal("empty root key must be rejected")
	}
}

func TestDeriveKeyringIsStableAndPurposeSeparated(t *testing.T) {
	first, err := DeriveKeyring("test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	second, err := DeriveKeyring("test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Audit, second.Audit) || !bytes.Equal(first.Share, second.Share) {
		t.Fatal("same root key must derive stable purpose keys")
	}
	if bytes.Equal(first.Audit, first.Share) {
		t.Fatal("audit and share keys must be purpose-separated")
	}
}

func TestSettingsCipherKeepsStoredCiphertextsReadableWithRootKey(t *testing.T) {
	const rootKey = "existing-settings-encryption-key"
	cipher := NewSettingsCipher(rootKey)
	stored, err := cipher.Encrypt("captcha-secret")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := NewSettingsCipher(rootKey).Decrypt(stored)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "captcha-secret" {
		t.Fatalf("decrypted value = %q, want original settings value", plain)
	}
}
