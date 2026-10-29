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
	}{{"default", []string{"subflow", "serve"}, "", []string{"subflow", "serve", "--https=0.0.0.0:8080", "--http=127.0.0.1:0"}, false}, {"environment", []string{"subflow", "serve"}, "9090", []string{"subflow", "serve", "--https=0.0.0.0:9090", "--http=127.0.0.1:0"}, false}, {"explicit HTTPS address", []string{"subflow", "serve", "--https=127.0.0.1:7000"}, "bad", []string{"subflow", "serve", "--https=127.0.0.1:7000", "--http=127.0.0.1:0"}, false}, {"explicit loopback HTTP is retained", []string{"subflow", "serve", "--http", "127.0.0.1:7000"}, "9090", []string{"subflow", "serve", "--http", "127.0.0.1:7000", "--https=0.0.0.0:9090"}, false}, {"explicit public HTTP is rejected", []string{"subflow", "serve", "--http=0.0.0.0:7000"}, "9090", nil, true}, {"other command", []string{"subflow", "superuser", "list"}, "bad", []string{"subflow", "superuser", "list"}, false}, {"invalid", []string{"subflow", "serve"}, "70000", nil, true}}
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
	if got := AppURL("", "9090"); got != "https://localhost:9090" {
		t.Fatal(got)
	}
	if got := AppURL("https://subflow.example/", "8080"); got != "https://subflow.example" {
		t.Fatal(got)
	}
}

func TestServeHTTPSAddress(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
		bad  bool
	}{
		{name: "configured flag", args: []string{"subflow", "serve", "--https", "0.0.0.0:8443"}, want: "0.0.0.0:8443"},
		{name: "default", args: []string{"subflow", "serve"}, want: "0.0.0.0:8080"},
		{name: "invalid empty", args: []string{"subflow", "serve", "--https="}, bad: true},
		{name: "invalid port", args: []string{"subflow", "serve", "--https=:0"}, bad: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ServeHTTPSAddress(test.args)
			if (err != nil) != test.bad {
				t.Fatalf("ServeHTTPSAddress() error = %v", err)
			}
			if !test.bad && got != test.want {
				t.Fatalf("ServeHTTPSAddress() = %q, want %q", got, test.want)
			}
		})
	}
}
