package captcha

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	altcha "github.com/altcha-org/altcha-lib-go"
)

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func turnstileVerifier(status int, body string) Verifier {
	return Verifier{Client: &http.Client{Transport: roundTripper(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
	})}}
}

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

	if err = v.Verify(context.Background(), "altcha_community", secret, "", token, "", "", ""); err != nil {
		t.Fatalf("expected the solved payload to verify, got %v", err)
	}
}

func TestCommunityChallengeRejectsGarbageToken(t *testing.T) {
	v := NewVerifier()
	if err := v.Verify(context.Background(), "altcha_community", "test-secret", "", "not-a-real-token", "", "", ""); err == nil {
		t.Fatal("expected an error for a garbage token, not a panic or a pass")
	}
}

func TestVerifyNoopsForEmptyProvider(t *testing.T) {
	v := NewVerifier()
	if err := v.Verify(context.Background(), "", "any-secret", "", "any-token", "", "", ""); err != nil {
		t.Fatalf("expected no captcha configured to be a no-op, got %v", err)
	}
}

func TestTurnstileRequiresMatchingActionAndApplicationURLHostname(t *testing.T) {
	cases := []struct {
		name, action, hostname, appURL, wantReason string
	}{
		{"success", "login", "subflow.example.com", "https://subflow.example.com", ""},
		{"wrong action", "register", "subflow.example.com", "https://subflow.example.com", "action_mismatch"},
		{"wrong hostname", "login", "other.example.com", "https://subflow.example.com", "hostname_mismatch"},
		{"missing configured hostname", "login", "subflow.example.com", "", "hostname_not_configured"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"success":true,"action":"` + tc.action + `","hostname":"` + tc.hostname + `"}`
			err := turnstileVerifier(http.StatusOK, body).Verify(context.Background(), "turnstile", "secret", "", "token", "203.0.113.1", "login", tc.appURL)
			if tc.wantReason == "" && err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if tc.wantReason != "" && FailureReason(err) != tc.wantReason {
				t.Fatalf("FailureReason() = %q, want %q", FailureReason(err), tc.wantReason)
			}
		})
	}
}

func TestTurnstileClassifiesProviderFailuresWithoutLeakingResponse(t *testing.T) {
	tests := []struct {
		name     string
		verifier Verifier
		want     string
	}{
		{"replayed token", turnstileVerifier(http.StatusOK, `{"success":false,"error-codes":["timeout-or-duplicate"]}`), "timeout_or_duplicate"},
		{"invalid secret", turnstileVerifier(http.StatusOK, `{"success":false,"error-codes":["invalid-input-secret"]}`), "secret_invalid"},
		{"non success HTTP", turnstileVerifier(http.StatusBadGateway, `{"success":false}`), "provider_response_invalid"},
		{"network error", Verifier{Client: &http.Client{Transport: roundTripper(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })}}, "provider_unavailable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.verifier.Verify(context.Background(), "turnstile", "secret", "", "token", "", "login", "https://subflow.example.com")
			if got := FailureReason(err); got != tc.want {
				t.Fatalf("FailureReason() = %q, want %q", got, tc.want)
			}
		})
	}
}
