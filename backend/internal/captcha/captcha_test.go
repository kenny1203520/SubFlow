package captcha

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	altcha "github.com/altcha-org/altcha-lib-go"
)

// Reproduces the exact failure the user reported: the backend must issue a
// classic-shaped ALTCHA challenge and accept a classic-shaped solved payload,
// since the frontend loads the public `altcha` CDN widget which only speaks
// that protocol. This is the first test this package has ever had.
func TestCommunityChallengeRoundTrips(t *testing.T) {
	v := NewVerifier()
	secret := "test-secret"

	challenge, err := v.CreateCommunityChallenge(secret, "authentication")
	if err != nil {
		t.Fatal(err)
	}
	if challenge.Algorithm == "" || challenge.Challenge == "" || challenge.Salt == "" || challenge.Signature == "" {
		t.Fatalf("expected a flat classic challenge shape, got %#v", challenge)
	}

	solution, err := altcha.SolveChallenge(challenge.Challenge, challenge.Salt, altcha.Algorithm(challenge.Algorithm), int(challenge.MaxNumber), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if solution == nil {
		t.Fatal("expected the challenge to be solvable")
	}

	payload := altcha.Payload{Algorithm: challenge.Algorithm, Challenge: challenge.Challenge, Number: int64(solution.Number), Salt: challenge.Salt, Signature: challenge.Signature}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	token := base64.StdEncoding.EncodeToString(encoded)

	if err = v.Verify(context.Background(), "altcha_community", secret, "", token, ""); err != nil {
		t.Fatalf("expected the solved payload to verify, got %v", err)
	}
}

func TestCommunityChallengeRejectsGarbageToken(t *testing.T) {
	v := NewVerifier()
	if err := v.Verify(context.Background(), "altcha_community", "test-secret", "", "not-a-real-token", ""); err == nil {
		t.Fatal("expected an error for a garbage token, not a panic or a pass")
	}
}

func TestVerifyNoopsForEmptyProvider(t *testing.T) {
	v := NewVerifier()
	if err := v.Verify(context.Background(), "", "any-secret", "", "any-token", ""); err != nil {
		t.Fatalf("expected no captcha configured to be a no-op, got %v", err)
	}
}
