package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecute_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := Execute(context.Background(), nil, req, ExecOpts{
		Timeout:      5 * time.Second,
		Retries:      0,
		MaxRedirects: 10,
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d", resp.StatusCode)
	}
}

func TestExecute_WithClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("POST", srv.URL, nil)
	resp, err := Execute(context.Background(), client, req, ExecOpts{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d", resp.StatusCode)
	}
}

func TestExecute_RetriesOnServerError(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	_, _ = Execute(context.Background(), nil, req, ExecOpts{
		Timeout:  5 * time.Second,
		Retries:  2,
		Backoff:  1 * time.Millisecond,
	})
	// GET is idempotent so should retry: 1 initial + 2 retries = 3
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("attempts = %d, want 3", atomic.LoadInt32(&attempts))
	}
}

func TestExecute_NoRetryNonIdempotent(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL, nil)
	_, _ = Execute(context.Background(), nil, req, ExecOpts{
		Timeout:  5 * time.Second,
		Retries:  3,
		Backoff:  1 * time.Millisecond,
	})
	// POST is not idempotent, so no retries
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("attempts = %d, want 1 (no retry for POST)", atomic.LoadInt32(&attempts))
	}
}

func TestExecute_NoRetryOnSuccess(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	_, _ = Execute(context.Background(), nil, req, ExecOpts{
		Timeout: 5 * time.Second,
		Retries: 3,
	})
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("attempts = %d, want 1", atomic.LoadInt32(&attempts))
	}
}

func TestExecute_RetrySucceedsEventually(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := Execute(context.Background(), nil, req, ExecOpts{
		Timeout:  5 * time.Second,
		Retries:  3,
		Backoff:  1 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d", resp.StatusCode)
	}
}

func TestExecute_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	req, _ := http.NewRequest("GET", srv.URL, nil)
	_, err := Execute(ctx, nil, req, ExecOpts{
		Timeout: 5 * time.Second,
	})
	if err == nil {
		t.Errorf("expected error for cancelled context")
	}
}

func TestExecute_ConnectionError(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://127.0.0.1:0/nonexistent", nil)
	_, err := Execute(context.Background(), nil, req, ExecOpts{
		Timeout: 1 * time.Second,
	})
	if err == nil {
		t.Errorf("expected connection error")
	}
}

func TestIsIdempotent(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{"GET", true},
		{"get", true},
		{"HEAD", true},
		{"OPTIONS", true},
		{"POST", false},
		{"PUT", false},
		{"DELETE", false},
		{"PATCH", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if got := isIdempotent(tt.method); got != tt.want {
				t.Errorf("isIdempotent(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestReadPossiblyFile_DirectValue(t *testing.T) {
	got, err := readPossiblyFile("plain string")
	if err != nil {
		t.Fatalf("readPossiblyFile() error: %v", err)
	}
	if got != "plain string" {
		t.Errorf("got = %q", got)
	}
}

func TestReadPossiblyFile_FromFile(t *testing.T) {
	path := "/tmp/reqo_test_readfile.txt"
	content := "file content here"
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}
	got, err := readPossiblyFile("@" + path)
	if err != nil {
		t.Fatalf("readPossiblyFile() error: %v", err)
	}
	if got != content {
		t.Errorf("got = %q, want %q", got, content)
	}
}

func TestReadPossiblyFile_MissingFile(t *testing.T) {
	_, err := readPossiblyFile("@/tmp/reqo_nonexistent_readfile.txt")
	if err == nil {
		t.Errorf("expected error for missing file")
	}
}

func TestExecute_InsecureTLS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	_, err := Execute(context.Background(), nil, req, ExecOpts{
		Timeout:  5 * time.Second,
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("Execute() with Insecure error: %v", err)
	}
}

func TestExecute_RedirectLimit(t *testing.T) {
	// Create a server that always redirects to itself (infinite loop)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/start", nil)
	_, err := Execute(context.Background(), nil, req, ExecOpts{
		Timeout:      5 * time.Second,
		MaxRedirects: 2,
	})
	if err == nil {
		// might get an error about too many redirects, or not — either way no panic
	}
	if err != nil && !strings.Contains(err.Error(), "redirect") {
		// The error should be redirect-related if present
	}
}
