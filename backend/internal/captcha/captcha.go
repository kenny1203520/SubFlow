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

func (v Verifier) Verify(ctx context.Context, provider, secret, verifyURL, token, remoteIP string) error {
	if provider == "" { return nil }
	if secret == "" || token == "" { return errors.New("captcha token is required") }
	switch provider {
	case "altcha": fallthrough // legacy setting compatibility
	case "altcha_community":
		ok, err := altcha.VerifySolution(token, secret, true)
		if err != nil || !ok { return errors.New("captcha verification failed") }
		return nil
	case "altcha_sentinel":
		if verifyURL == "" { return errors.New("altcha sentinel verify url is required") }
		result, err := altchav2.VerifyServer(ctx, altchav2.VerifyServerOptions{URL: verifyURL, Payload: token, Secret: secret, HTTPClient: v.Client})
		if err != nil || !result.Verified { return errors.New("captcha verification failed") }
		return nil
	}
	endpoint := map[string]string{"recaptcha": "https://www.google.com/recaptcha/api/siteverify", "turnstile": "https://challenges.cloudflare.com/turnstile/v0/siteverify", "hcaptcha": "https://hcaptcha.com/siteverify"}[provider]
	if endpoint == "" { return errors.New("unsupported captcha provider") }
	form := url.Values{"secret": {secret}, "response": {token}}
	if remoteIP != "" { form.Set("remoteip", remoteIP) }
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil { return err }
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := v.Client.Do(req); if err != nil { return err }; defer resp.Body.Close()
	var result struct { Success bool `json:"success"` }
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil { return err }
	if resp.StatusCode/100 != 2 || !result.Success { return errors.New("captcha verification failed") }
	return nil
}
