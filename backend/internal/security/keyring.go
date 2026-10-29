package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

// Keyring holds purpose-separated keys derived from the deployment's existing
// settings-encryption key. It never contains a persistent copy of the root key.
type Keyring struct {
	Audit []byte
	Share []byte
}

func DeriveKeyring(root string) (Keyring, error) {
	if root == "" {
		return Keyring{}, errors.New("SUBFLOW_SETTINGS_ENCRYPTION_KEY is required")
	}
	return Keyring{
		Audit: deriveSubkey(root, "subflow/audit/v1"),
		Share: deriveSubkey(root, "subflow/share-cookie/v1"),
	}, nil
}

func deriveSubkey(root, purpose string) []byte {
	// HKDF-SHA256 with a fixed application salt and a purpose-specific info
	// string keeps keys independent while allowing one operator-managed root.
	extract := hmac.New(sha256.New, []byte("SubFlow/HKDF-SHA256/v1"))
	_, _ = extract.Write([]byte(root))
	prk := extract.Sum(nil)
	expand := hmac.New(sha256.New, prk)
	_, _ = expand.Write([]byte(purpose))
	_, _ = expand.Write([]byte{1})
	return expand.Sum(nil)
}
