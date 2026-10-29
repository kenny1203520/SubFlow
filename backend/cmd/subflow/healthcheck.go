package main

import (
	"fmt"
	"net/http"
	"time"
)

func checkHealth(endpoint string, configured ...*http.Client) error {
	client := &http.Client{Timeout: 2 * time.Second}
	if len(configured) > 0 && configured[0] != nil {
		client = configured[0]
	}
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
