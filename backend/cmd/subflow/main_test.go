package main

import (
	"strings"
	"testing"
)

func TestSetupStartupNotice(t *testing.T) {
	if got := setupStartupNotice(""); got != "" {
		t.Fatalf("empty link produced notice %q", got)
	}
	link := "http://localhost:5173/setup?token=one-time-token"
	notice := setupStartupNotice(link)
	for _, want := range []string{"FIRST-RUN SETUP REQUIRED", link, "\033[1;97;42m"} {
		if !strings.Contains(notice, want) {
			t.Fatalf("notice missing %q: %q", want, notice)
		}
	}
}
