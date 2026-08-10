package captcha

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	altcha "github.com/altcha-org/altcha-lib-go/v2"
)

type Verifier struct{ Client *http.Client }

func NewVerifier() Verifier { return Verifier{Client: &http.Client{Timeout: 8 * time.Second}} }

// CreateCommunityChallenge creates a short-lived, signed ALTCHA Community v2
// challenge. The caller exposes it through its own public endpoint.
func (v Verifier) CreateCommunityChallenge(secret, flow string) (altcha.Challenge, error) {
	if secret == "" { return altcha.Challenge{}, errors.New("altcha secret is required") }
	expires := time.Now().Add(5 * time.Minute)
	return altcha.CreateChallenge(altcha.CreateChallengeOptions{Algorithm: "SHA-256", Cost: 10000, ExpiresAt: &expires, Data: map[string]interface{}{"flow": flow}, HMACSignatureSecret: secret})
}

func decodePayload(token string) (altcha.Payload, error) {
	var payload altcha.Payload
	if json.Unmarshal([]byte(token), &payload) == nil { return payload, nil }
	for _, encoding := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		decoded, err := encoding.DecodeString(token)
		if err == nil && json.Unmarshal(decoded, &payload) == nil { return payload, nil }
	}
	return payload, errors.New("invalid altcha payload")
}

func (v Verifier) Verify(ctx context.Context, provider, secret, verifyURL, token, remoteIP string) error {
	if provider == "" { return nil }
	if secret == "" || token == "" { return errors.New("captcha token is required") }
	switch provider {
	case "altcha": provider = "altcha_community" // legacy setting compatibility
	case "altcha_community":
		payload, err := decodePayload(token); if err != nil { return err }
		result, err := altcha.VerifySolution(altcha.VerifySolutionOptions{Challenge: payload.Challenge, Solution: payload.Solution, HMACSignatureSecret: secret})
		if err != nil || !result.Verified { return errors.New("captcha verification failed") }
		return nil
	case "altcha_sentinel":
		if verifyURL == "" { return errors.New("altcha sentinel verify url is required") }
		result, err := altcha.VerifyServer(ctx, altcha.VerifyServerOptions{URL: verifyURL, Payload: token, Secret: secret, HTTPClient: v.Client})
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
