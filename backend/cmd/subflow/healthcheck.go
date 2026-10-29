package main

import (
	"fmt"
	"net/http"
	"time"
)

func checkHealth(endpoint string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("readiness check failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("readiness check returned HTTP %d", response.StatusCode)
	}
	return nil
}
