package captcha

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Verifier struct{ Client *http.Client }

func NewVerifier() Verifier { return Verifier{Client: &http.Client{Timeout: 8 * time.Second}} }

func (v Verifier) Verify(ctx context.Context, provider, secret, token, remoteIP string) error {
	if provider == "" {
		return nil
	}
	if secret == "" || token == "" {
		return errors.New("captcha token is required")
	}
	if provider == "altcha" {
		return errors.New("altcha verification must be configured by the deployment")
	}
	endpoint := map[string]string{"recaptcha": "https://www.google.com/recaptcha/api/siteverify", "turnstile": "https://challenges.cloudflare.com/turnstile/v0/siteverify", "hcaptcha": "https://hcaptcha.com/siteverify"}[provider]
	if endpoint == "" {
		return errors.New("unsupported captcha provider")
	}
	form := url.Values{"secret": {secret}, "response": {token}}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := v.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result struct {
		Success bool `json:"success"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 || !result.Success {
		return errors.New("captcha verification failed")
	}
	return nil
}
