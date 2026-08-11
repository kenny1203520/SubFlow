package config

import (
	"reflect"
	"testing"
)

func TestWithServeAddress(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		port    string
		want    []string
		wantErr bool
	}{{"default", []string{"subflow", "serve"}, "", []string{"subflow", "serve", "--http=0.0.0.0:8080"}, false}, {"environment", []string{"subflow", "serve"}, "9090", []string{"subflow", "serve", "--http=0.0.0.0:9090"}, false}, {"explicit equals wins", []string{"subflow", "serve", "--http=127.0.0.1:7000"}, "9090", []string{"subflow", "serve", "--http=127.0.0.1:7000"}, false}, {"explicit pair wins", []string{"subflow", "serve", "--http", "127.0.0.1:7000"}, "9090", []string{"subflow", "serve", "--http", "127.0.0.1:7000"}, false}, {"other command", []string{"subflow", "superuser", "list"}, "bad", []string{"subflow", "superuser", "list"}, false}, {"invalid", []string{"subflow", "serve"}, "70000", nil, true}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := WithServeAddress(tc.args, tc.port)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v", err)
			}
			if !tc.wantErr && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v want %#v", got, tc.want)
			}
		})
	}
}
func TestAppURL(t *testing.T) {
	if got := AppURL("", "9090"); got != "http://localhost:9090" {
		t.Fatal(got)
	}
	if got := AppURL("https://subflow.example/", "8080"); got != "https://subflow.example" {
		t.Fatal(got)
	}
}

func TestPublicAppURL(t *testing.T) {
	cases := []struct {
		name, configured, port, environment, want string
		wantErr                                   bool
	}{
		{"development fallback", "", "5173", "development", "http://localhost:5173", false},
		{"production requires public URL", "", "8080", "production", "", true},
		{"production public URL", "https://subflow.example/", "8080", "production", "https://subflow.example", false},
		{"invalid URL", "subflow.example", "8080", "production", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PublicAppURL(tc.configured, tc.port, tc.environment)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("PublicAppURL() = %q, %v; want %q, error=%v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}
