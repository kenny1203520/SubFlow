package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const DefaultHTTPPort = "8080"

// WithServeAddress gives PocketBase a public HTTPS listener and an ephemeral
// loopback-only HTTP redirect listener. Public HTTP listeners are rejected.
func WithServeAddress(args []string, configuredPort string) ([]string, error) {
	if len(args) < 2 || args[1] != "serve" {
		return args, nil
	}
	result := append([]string(nil), args...)
	port := strings.TrimSpace(configuredPort)
	if port == "" {
		port = DefaultHTTPPort
	}
	if value, found, err := flagValue(result, "--https"); err != nil {
		return nil, err
	} else if found {
		if err = validateAddress(value, false); err != nil {
			return nil, fmt.Errorf("invalid PocketBase HTTPS address")
		}
	} else {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return nil, fmt.Errorf("invalid SUBFLOW_HTTP_PORT %q: expected 1-65535", configuredPort)
		}
		result = append(result, "--https=0.0.0.0:"+port)
	}
	if value, found, err := flagValue(result, "--http"); err != nil {
		return nil, err
	} else if found {
		if err = validateLoopbackAddress(value); err != nil {
			return nil, fmt.Errorf("SubFlow only permits a loopback HTTP redirect listener; use HTTPS for external traffic")
		}
	} else {
		result = append(result, "--http=127.0.0.1:0")
	}
	return result, nil
}

func flagValue(args []string, name string) (string, bool, error) {
	for index := 2; index < len(args); index++ {
		if args[index] == name {
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "--") {
				return "", true, fmt.Errorf("%s requires a TCP address", name)
			}
			return args[index+1], true, nil
		}
		if strings.HasPrefix(args[index], name+"=") {
			value := strings.TrimPrefix(args[index], name+"=")
			if value == "" {
				return "", true, fmt.Errorf("%s requires a TCP address", name)
			}
			return value, true, nil
		}
	}
	return "", false, nil
}

func validateAddress(value string, allowZero bool) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil || host == "" {
		return fmt.Errorf("invalid TCP address")
	}
	number, err := strconv.Atoi(port)
	minimum := 1
	if allowZero {
		minimum = 0
	}
	if err != nil || number < minimum || number > 65535 {
		return fmt.Errorf("invalid TCP port")
	}
	return nil
}

func validateLoopbackAddress(value string) error {
	if err := validateAddress(value, true); err != nil {
		return err
	}
	host, _, _ := net.SplitHostPort(value)
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("HTTP listener must be loopback-only")
	}
	return nil
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

func ServeHTTPSAddress(args []string) (string, error) {
	value, found, err := flagValue(args, "--https")
	if err != nil {
		return "", fmt.Errorf("invalid PocketBase HTTPS address")
	}
	if !found {
		value = "0.0.0.0:" + DefaultHTTPPort
	}
	if err := validateAddress(value, false); err != nil {
		return "", fmt.Errorf("invalid PocketBase HTTPS address")
	}
	return value, nil
}

func AppURL(configured, port string) string {
	if configured != "" {
		return strings.TrimRight(configured, "/")
	}
	if strings.TrimSpace(port) == "" {
		port = DefaultHTTPPort
	}
	return "https://localhost:" + port
}
