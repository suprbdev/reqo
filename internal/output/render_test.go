package output

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func newResp(t *testing.T, status int, headers map[string]string, body string) *http.Response {
	t.Helper()
	resp := &http.Response{
		Status:     http.StatusText(status),
		StatusCode: status,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Body:       nopCloser{strings.NewReader(body)},
	}
	for k, v := range headers {
		resp.Header.Set(k, v)
	}
	return resp
}

func TestRender_PlainBody(t *testing.T) {
	resp := newResp(t, 200, map[string]string{"Content-Type": "text/plain"}, "hello world")
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestRender_JSONPretty(t *testing.T) {
	raw := `{"name":"test","age":30}`
	resp := newResp(t, 200, map[string]string{"Content-Type": "application/json"}, raw)
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "  \"name\":") {
		t.Errorf("JSON should be pretty-printed with indentation: %q", out)
	}
}

func TestRender_RawOutput(t *testing.T) {
	raw := `{"name":"test"}`
	resp := newResp(t, 200, map[string]string{"Content-Type": "application/json"}, raw)
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{RawOutput: true}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if buf.String() != raw {
		t.Errorf("raw output = %q, want %q", buf.String(), raw)
	}
}

func TestRender_ShowHeaders(t *testing.T) {
	resp := newResp(t, 200, map[string]string{
		"Content-Type":   "application/json",
		"X-Custom-Header": "customval",
	}, `{"ok":true}`)
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{ShowHeaders: true}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "HTTP/1.1 200") {
		t.Errorf("should show HTTP status line: %q", out)
	}
	if !strings.Contains(out, "Content-Type:") {
		t.Errorf("should show Content-Type header: %q", out)
	}
	if !strings.Contains(out, "X-Custom-Header:") {
		t.Errorf("should show X-Custom-Header: %q", out)
	}
}

func TestRender_EmptyBody(t *testing.T) {
	resp := newResp(t, 204, map[string]string{}, "")
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if buf.String() != "" {
		t.Errorf("empty body output should be empty, got %q", buf.String())
	}
}

func TestRender_BodyWithoutNewline(t *testing.T) {
	resp := newResp(t, 200, map[string]string{"Content-Type": "text/plain"}, "no newline")
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Errorf("output should end with newline")
	}
}

func TestRender_BodyWithNewline(t *testing.T) {
	resp := newResp(t, 200, map[string]string{"Content-Type": "text/plain"}, "has newline\n")
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if strings.Count(buf.String(), "\n") != 1 {
		t.Errorf("output should have exactly one newline, got: %q", buf.String())
	}
}

func TestRender_InvalidJSON(t *testing.T) {
	resp := newResp(t, 200, map[string]string{"Content-Type": "application/json"}, "{invalid json}")
	var buf bytes.Buffer
	// Should not error, just output raw body since indent fails
	if err := Render(resp, &buf, RenderOpts{}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(buf.String(), "{invalid json}") {
		t.Errorf("should output raw invalid JSON: %q", buf.String())
	}
}

func TestRender_JQExpr(t *testing.T) {
	raw := `{"name":"test","age":30,"city":"NYC"}`
	resp := newResp(t, 200, map[string]string{"Content-Type": "application/json"}, raw)
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{JQExpr: ".name"}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "test") {
		t.Errorf("jq .name should output 'test': %q", out)
	}
}

func TestRender_JQExpr_Array(t *testing.T) {
	raw := `[{"id":1},{"id":2},{"id":3}]`
	resp := newResp(t, 200, map[string]string{"Content-Type": "application/json"}, raw)
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{JQExpr: ".[].id"}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "1") || !strings.Contains(out, "2") || !strings.Contains(out, "3") {
		t.Errorf("jq .[].id should contain 1,2,3: %q", out)
	}
}

func TestRender_JQExpr_InvalidExpr(t *testing.T) {
	resp := newResp(t, 200, map[string]string{"Content-Type": "application/json"}, `{"a":1}`)
	var buf bytes.Buffer
	err := Render(resp, &buf, RenderOpts{JQExpr: "...."})
	if err == nil {
		t.Errorf("expected error for invalid jq expression")
	}
}

func TestRender_JQExpr_NonJSONBody(t *testing.T) {
	resp := newResp(t, 200, map[string]string{"Content-Type": "text/plain"}, "not json")
	var buf bytes.Buffer
	err := Render(resp, &buf, RenderOpts{JQExpr: ".a"})
	if err == nil {
		t.Errorf("expected error when applying jq to non-JSON")
	}
}

func TestRender_JSONArrayPretty(t *testing.T) {
	raw := `[1,2,3]`
	resp := newResp(t, 200, map[string]string{"Content-Type": "application/json"}, raw)
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "  1,") {
		t.Errorf("JSON array should be pretty-printed: %q", out)
	}
}

func TestRender_StatusLine(t *testing.T) {
	resp := newResp(t, 404, map[string]string{}, "not found")
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{ShowHeaders: true}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "404") {
		t.Errorf("should contain status code 404: %q", out)
	}
}

func TestRender_HTTPVersion(t *testing.T) {
	resp := newResp(t, 200, map[string]string{}, "ok")
	resp.ProtoMajor = 2
	resp.ProtoMinor = 0
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{ShowHeaders: true}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "HTTP/2.0") {
		t.Errorf("should show HTTP/2.0 version: %q", out)
	}
}

func TestRender_MultipleHeaders(t *testing.T) {
	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Body:       nopCloser{strings.NewReader("")},
	}
	resp.Header.Add("Set-Cookie", "a=1")
	resp.Header.Add("Set-Cookie", "b=2")
	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{ShowHeaders: true}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()
	if strings.Count(out, "Set-Cookie:") != 2 {
		t.Errorf("should show 2 Set-Cookie headers: %q", out)
	}
}

func TestRender_ServerResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if err := Render(resp, &buf, RenderOpts{ShowHeaders: true}); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(buf.String(), "ok") {
		t.Errorf("output should contain response body: %q", buf.String())
	}
}
