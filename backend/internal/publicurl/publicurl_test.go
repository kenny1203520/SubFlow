package publicurl

import "testing"

func TestParseAcceptsHTTPSOrigins(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "HTTPS with root slash and whitespace", input: "  https://subflow.example.test/  ", want: "https://subflow.example.test"},
		{name: "localhost with custom port", input: "https://localhost:8080/", want: "https://localhost:8080"},
		{name: "internal hostname", input: "https://subflow-internal:8090", want: "https://subflow-internal:8090"},
		{name: "IPv4", input: "https://192.0.2.10:8443", want: "https://192.0.2.10:8443"},
		{name: "IPv6", input: "https://[2001:db8::1]:8080", want: "https://[2001:db8::1]:8080"},
		{name: "normalizes scheme casing", input: "HTTPS://subflow.example.test", want: "https://subflow.example.test"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := Parse(test.input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", test.input, err)
			}
			if got := parsed.String(); got != test.want {
				t.Fatalf("Parse(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestParseRejectsUnsupportedURLForms(t *testing.T) {
	for _, input := range []string{
		"", "   ", "subflow.example.test", "//subflow.example.test", "ftp://subflow.example.test", "http://subflow.example.test",
		"https:///missing-host", "https://user:password@subflow.example.test", "https://subflow.example.test?",
		"https://subflow.example.test?next=/", "https://subflow.example.test#", "https://subflow.example.test#invite",
		"https://subflow.example.test/app", "https://subflow.example.test/app/", "https://subflow.example.test:0",
		"https://subflow.example.test:65536", "https://subflow.example.test:", "https://bad_host.example.test", "https://-bad.example.test",
	} {
		t.Run(input, func(t *testing.T) {
			if parsed, err := Parse(input); err == nil {
				t.Fatalf("Parse(%q) = %q, want an error", input, parsed)
			}
		})
	}
}
