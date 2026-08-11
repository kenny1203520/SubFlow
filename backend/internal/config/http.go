package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const DefaultHTTPPort = "8080"

// WithServeAddress injects PocketBase's --http flag only for the serve command.
// An explicit CLI --http always wins over SUBFLOW_HTTP_PORT.
func WithServeAddress(args []string, configuredPort string) ([]string, error) {
	if len(args) < 2 || args[1] != "serve" {
		return args, nil
	}
	for i, arg := range args[2:] {
		if arg == "--http" || strings.HasPrefix(arg, "--http=") {
			_ = i
			return args, nil
		}
	}
	port := strings.TrimSpace(configuredPort)
	if port == "" {
		port = DefaultHTTPPort
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return nil, fmt.Errorf("invalid SUBFLOW_HTTP_PORT %q: expected 1-65535", configuredPort)
	}
	result := append([]string(nil), args...)
	return append(result, "--http=0.0.0.0:"+port), nil
}

func AppURL(configured, port string) string {
	if configured != "" {
		return strings.TrimRight(configured, "/")
	}
	if strings.TrimSpace(port) == "" {
		port = DefaultHTTPPort
	}
	return "http://localhost:" + port
}

// PublicAppURL resolves the browser-facing URL used in emails and one-time
// setup links. Production must never silently emit an unusable localhost URL.
func PublicAppURL(configured, port, environment string) (string, error) {
	value := strings.TrimRight(strings.TrimSpace(configured), "/")
	if value == "" {
		if strings.EqualFold(strings.TrimSpace(environment), "production") {
			return "", fmt.Errorf("SUBFLOW_APP_URL is required when SUBFLOW_ENV=production; set it to the public HTTPS URL served by Nginx")
		}
		return AppURL("", port), nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid SUBFLOW_APP_URL %q: expected an absolute http(s) URL", configured)
	}
	return value, nil
}
