package config

import (
	"fmt"
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
