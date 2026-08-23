package captcha

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	altcha "github.com/altcha-org/altcha-lib-go"
	altchav2 "github.com/altcha-org/altcha-lib-go/v2"
)

type Verifier struct{ Client *http.Client }

// ApplicationIdentity is sourced from PocketBase's Application settings.  It
// deliberately has no environment-variable fallback: CAPTCHA providers must
// be configured for the same public application that PocketBase advertises.
// Name is retained here for providers that identify an application by name;
// the currently supported browser CAPTCHA providers bind to URL or site key.
type ApplicationIdentity struct {
	Name    string
	URL     string
	SiteKey string
}

// VerificationError carries a safe, stable failure category for audit logs.
// It deliberately excludes provider response text, submitted tokens and keys.
type VerificationError struct{ Reason string }

func (e *VerificationError) Error() string { return "captcha verification failed" }

func failure(reason string) error { return &VerificationError{Reason: reason} }

// FailureReason returns a non-sensitive category suitable for audit summaries.
func FailureReason(err error) string {
	var verificationErr *VerificationError
	if errors.As(err, &verificationErr) {
		return verificationErr.Reason
	}
	return "verification_failed"
}

func NewVerifier() Verifier { return Verifier{Client: &http.Client{Timeout: 8 * time.Second}} }

// CreateCommunityChallenge creates a short-lived, signed classic ALTCHA
// challenge (flat algorithm/challenge/salt/maxNumber/signature JSON). This
// must stay on the classic protocol because the frontend loads the public
// `altcha` CDN widget, which only understands that shape — the newer KDF v2
// protocol (altcha-lib-go/v2) produces a differently-shaped challenge the
// widget can't parse, which is what caused every Community verification to
// fail with the widget's own generic error.
func (v Verifier) CreateCommunityChallenge(secret, flow string) (altcha.Challenge, error) {
	if secret == "" {
		return altcha.Challenge{}, errors.New("altcha secret is required")
	}
	expires := time.Now().Add(5 * time.Minute)
	params := url.Values{"flow": {flow}}
	return altcha.CreateChallenge(altcha.ChallengeOptions{Algorithm: altcha.SHA256, HMACKey: secret, Expires: &expires, Params: params})
}

// Verify validates a provider response. The application identity always comes
// from PocketBase's Application settings. Turnstile validates its action and
// hostname; reCAPTCHA validates its hostname; hCaptcha receives the configured
// site key, as recommended by its Siteverify contract. hCaptcha explicitly
// documents its hostname as non-authenticating metadata, so it is not trusted
// as an authorization signal here.
func (v Verifier) Verify(ctx context.Context, provider, secret, verifyURL, token, remoteIP, expectedAction string, app ApplicationIdentity) error {
	if provider == "" {
		return nil
	}
	if secret == "" {
		return failure("secret_missing")
	}
	if token == "" {
		return failure("token_missing")
	}
	switch provider {
	case "altcha":
		fallthrough // legacy setting compatibility
	case "altcha_community":
		ok, err := altcha.VerifySolution(token, secret, true)
		if err != nil || !ok {
			return failure("verification_failed")
		}
		return nil
	case "altcha_sentinel":
		if verifyURL == "" {
			return failure("verification_url_missing")
		}
		result, err := altchav2.VerifyServer(ctx, altchav2.VerifyServerOptions{URL: verifyURL, Payload: token, Secret: secret, HTTPClient: v.Client})
		if err != nil || !result.Verified {
			return failure("verification_failed")
		}
		return nil
	}
	endpoint := map[string]string{"recaptcha": "https://www.google.com/recaptcha/api/siteverify", "turnstile": "https://challenges.cloudflare.com/turnstile/v0/siteverify", "hcaptcha": "https://api.hcaptcha.com/siteverify"}[provider]
	if endpoint == "" {
		return failure("provider_unsupported")
	}
	form := url.Values{"secret": {secret}, "response": {token}}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	if provider == "hcaptcha" {
		if app.SiteKey == "" {
			return failure("sitekey_not_configured")
		}
		form.Set("sitekey", app.SiteKey)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return failure("request_invalid")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := v.Client.Do(req)
	if err != nil {
		return failure("provider_unavailable")
	}
	defer resp.Body.Close()
	var result struct {
		Success    bool     `json:"success"`
		Action     string   `json:"action"`
		Hostname   string   `json:"hostname"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return failure("provider_response_invalid")
	}
	if resp.StatusCode/100 != 2 {
		return failure("provider_response_invalid")
	}
	if !result.Success {
		for _, code := range result.ErrorCodes {
			if code == "timeout-or-duplicate" || code == "expired-input-response" || code == "already-seen-response" {
				return failure("timeout_or_duplicate")
			}
			if code == "invalid-input-secret" || code == "sitekey-secret-mismatch" {
				return failure("secret_invalid")
			}
			if code == "invalid-input-response" || code == "missing-input-response" {
				return failure("token_invalid")
			}
		}
		return failure("verification_failed")
	}
	if provider == "turnstile" {
		if expectedAction == "" {
			return failure("action_not_configured")
		}
		if result.Action != expectedAction {
			return failure("action_mismatch")
		}
	}
	if provider != "turnstile" && provider != "recaptcha" {
		return nil
	}
	parsed, err := url.ParseRequestURI(app.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return failure("hostname_not_configured")
	}
	if !strings.EqualFold(result.Hostname, parsed.Hostname()) {
		return failure("hostname_mismatch")
	}
	return nil
}
