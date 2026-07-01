package httpx

import (
	"net/http"
	"strings"
	"testing"
)

func TestAsCurl_BasicGet(t *testing.T) {
	req, err := http.NewRequest("GET", "https://example.com/api", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := AsCurl(req)
	if err != nil {
		t.Fatalf("AsCurl() error: %v", err)
	}
	if !strings.HasPrefix(out, "curl -X GET") {
		t.Errorf("output should start with 'curl -X GET', got: %q", out)
	}
	if !strings.Contains(out, "https://example.com/api") {
		t.Errorf("output should contain URL: %q", out)
	}
}

func TestAsCurl_PostMethod(t *testing.T) {
	req, err := http.NewRequest("POST", "https://example.com/create", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := AsCurl(req)
	if !strings.HasPrefix(out, "curl -X POST") {
		t.Errorf("output should start with 'curl -X POST', got: %q", out)
	}
}

func TestAsCurl_WithHeaders(t *testing.T) {
	req, err := http.NewRequest("GET", "https://example.com/api", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-Custom", "value")
	out, _ := AsCurl(req)
	if !strings.Contains(out, `Authorization: Bearer token123`) {
		t.Errorf("output should contain Authorization header: %q", out)
	}
	if !strings.Contains(out, `X-Custom: value`) {
		t.Errorf("output should contain X-Custom header: %q", out)
	}
}

func TestAsCurl_WithBody(t *testing.T) {
	body := `{"name":"test"}`
	req, err := http.NewRequest("POST", "https://example.com/api", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	out, _ := AsCurl(req)
	if !strings.Contains(out, "--data-raw") {
		t.Errorf("output should contain --data-raw: %q", out)
	}
	if !strings.Contains(out, body) {
		t.Errorf("output should contain body: %q", out)
	}
}

func TestAsCurl_NoBody(t *testing.T) {
	req, err := http.NewRequest("GET", "https://example.com/api", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := AsCurl(req)
	if strings.Contains(out, "--data-raw") {
		t.Errorf("output should not contain --data-raw for GET without body: %q", out)
	}
}

func TestAsCurl_URLWithQuery(t *testing.T) {
	req, err := http.NewRequest("GET", "https://example.com/api?page=1&limit=10", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := AsCurl(req)
	if !strings.Contains(out, "page=1") || !strings.Contains(out, "limit=10") {
		t.Errorf("output should contain query params: %q", out)
	}
}

func TestAsCurl_QuoteInHeaderValue(t *testing.T) {
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Custom", `value with "quotes"`)
	out, _ := AsCurl(req)
	if !strings.Contains(out, `\"quotes\"`) {
		t.Errorf("output should escape quotes: %q", out)
	}
}

func TestShellQuote_EmptyString(t *testing.T) {
	got := shellQuote("")
	want := "''"
	if got != want {
		t.Errorf("shellQuote(\"\") = %q, want %q", got, want)
	}
}

func TestShellQuote_SimpleString(t *testing.T) {
	got := shellQuote("hello")
	want := "'hello'"
	if got != want {
		t.Errorf("shellQuote(\"hello\") = %q, want %q", got, want)
	}
}

func TestShellQuote_WithSingleQuote(t *testing.T) {
	got := shellQuote("it's")
	// shellQuote escapes ' as '\'' resulting in 'it'\''s'
	want := "'it'\\''s'"
	if got != want {
		t.Errorf("shellQuote(\"it's\") = %q, want %q", got, want)
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`no quotes`, `no quotes`},
		{`"quoted"`, `\"quoted\"`},
		{`"`, `\"`},
		{``, ``},
	}
	for _, tt := range tests {
		got := escape(tt.input)
		if got != tt.want {
			t.Errorf("escape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAsCurl_StripsUserinfo(t *testing.T) {
	req, err := http.NewRequest("GET", "https://user:pass@example.com/api", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := AsCurl(req)
	if strings.Contains(out, "user:pass") {
		t.Errorf("output should not contain userinfo: %q", out)
	}
}
