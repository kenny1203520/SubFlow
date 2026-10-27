package config

import (
	"fmt"
	"net"
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

func DataDirFromArgs(args []string) string {
	for index := 1; index < len(args); index++ {
		if strings.HasPrefix(args[index], "--dir=") {
			return strings.TrimPrefix(args[index], "--dir=")
		}
		if args[index] == "--dir" && index+1 < len(args) {
			return args[index+1]
		}
	}
	return "pb_data"
}

func ServeHTTPAddress(args []string) (string, bool, error) {
	value := "0.0.0.0:" + DefaultHTTPPort
	secure := false
	for index := 1; index < len(args); index++ {
		argument := args[index]
		if argument == "--https" || strings.HasPrefix(argument, "--https=") {
			secure = true
		}
		if strings.HasPrefix(argument, "--http=") {
			value = strings.TrimPrefix(argument, "--http=")
		}
		if argument == "--http" && index+1 < len(args) {
			value = args[index+1]
			index++
		}
	}
	if secure {
		return "", true, fmt.Errorf("SubFlow maintenance status requires HTTP behind a TLS-terminating proxy; configure --http and let the proxy handle HTTPS")
	}
	if _, _, err := net.SplitHostPort(value); err != nil {
		return "", false, fmt.Errorf("invalid PocketBase HTTP address")
	}
	return value, false, nil
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
