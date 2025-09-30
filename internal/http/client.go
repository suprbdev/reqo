package httpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/suprbdev/reqo/internal/project"
	"github.com/suprbdev/reqo/internal/template"
)

// RequestSpec aggregates all flags that can influence a request.
type RequestSpec struct {
	Method       string
	Path         string
	QueryParams  []string // each "k=v"
	Headers      []string // raw header lines
	UseHeaderSet string
	JSONBody     *string // raw JSON or @file
	RawBody      *string // raw body or @file
	FormFields   map[string]string
	Vars         map[string]string // --var values for expansion
	EnvName      string            // optional env override
}

// ExecOpts holds runtime options (timeout, retries,…)
type ExecOpts struct {
	Timeout      time.Duration
	Retries      int
	Backoff      time.Duration
	MaxRedirects int
	Insecure     bool
	CACert       string
}

// BuildRequest composes a *http.Request from the project, env and spec.
func BuildRequest(p *project.Project, spec RequestSpec) (*http.Request, error) {
	envName := spec.EnvName
	if envName == "" {
		envName = p.DefaultEnv
	}
	env, ok := p.Environments[envName]
	if !ok {
		return nil, fmt.Errorf("environment %q not defined", envName)
	}

	baseURL := template.Expand(env.BaseURL, spec.Vars)

	// 1️⃣ Resolve path + query
	path := template.Expand(spec.Path, spec.Vars)

	var fullURL string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		// Path is a full URL, use it directly
		fullURL = path
	} else {
		// Path is relative, combine with base URL
		fullURL = strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(path, "/")
	}

	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	q := u.Query()
	for _, qp := range spec.QueryParams {
		parts := strings.SplitN(qp, "=", 2)
		if len(parts) == 2 {
			q.Add(template.Expand(parts[0], spec.Vars), template.Expand(parts[1], spec.Vars))
		}
	}
	u.RawQuery = q.Encode()

	// 2️⃣ Body handling
	var body io.Reader
	contentType := ""

	if spec.JSONBody != nil && spec.RawBody != nil {
		return nil, fmt.Errorf("cannot specify both --json and --data")
	}

	switch {
	case spec.JSONBody != nil:
		data, err := readPossiblyFile(*spec.JSONBody)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader([]byte(data))
		contentType = "application/json"
	case spec.RawBody != nil:
		data, err := readPossiblyFile(*spec.RawBody)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(data)
	case len(spec.FormFields) > 0:
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		for k, v := range spec.FormFields {
			val := template.Expand(v, spec.Vars)
			if strings.HasPrefix(val, "@") { // file upload
				fpath := strings.TrimPrefix(val, "@")
				f, err := os.Open(fpath)
				if err != nil {
					return nil, fmt.Errorf("open form file %s: %w", fpath, err)
				}
				defer f.Close()
				part, err := w.CreateFormFile(k, filepath.Base(fpath))
				if err != nil {
					return nil, err
				}
				if _, err = io.Copy(part, f); err != nil {
					return nil, err
				}
			} else {
				_ = w.WriteField(k, val)
			}
		}
		w.Close()
		body = &b
		contentType = w.FormDataContentType()
	default:
		body = nil
	}

	method := spec.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	// 3️⃣ Merge headers (order of precedence):
	//    a) env.Headers
	//    b) header set referenced by spec.UseHeaderSet or call definition
	//    c) flags (--header)
	addHeaders := func(src []string) error {
		for _, line := range src {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid header %q – must be \"Key: Value\"", line)
			}
			key := strings.TrimSpace(parts[0])
			val := template.Expand(strings.TrimSpace(parts[1]), spec.Vars)
			req.Header.Add(key, val)
		}
		return nil
	}

	// env headers
	if err = addHeaders(env.Headers); err != nil {
		return nil, err
	}
	// header set (if any)
	if spec.UseHeaderSet != "" {
		if hs, ok := p.HeaderSets[spec.UseHeaderSet]; ok {
			if err = addHeaders(hs); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("header set %q not defined", spec.UseHeaderSet)
		}
	}
	// command‑line headers (override everything)
	if err = addHeaders(spec.Headers); err != nil {
		return nil, err
	}

	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

// Execute performs the HTTP request with timeout/retry/backoff.
func Execute(ctx context.Context, client *http.Client, req *http.Request, opts ExecOpts) (*http.Response, error) {
	if client == nil {
		tr := &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: opts.Insecure},
			MaxIdleConns:      10,
			DisableKeepAlives: false,
		}
		client = &http.Client{
			Transport: tr,
			Timeout:   opts.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > opts.MaxRedirects {
					return fmt.Errorf("stopped after %d redirects", opts.MaxRedirects)
				}
				return nil
			},
		}
	}

	var resp *http.Response
	var err error
	backoff := opts.Backoff

	for i := 0; i <= opts.Retries; i++ {
		resp, err = client.Do(req.WithContext(ctx))
		if err == nil && (resp.StatusCode < 500 || !isIdempotent(req.Method)) {
			break // success or non‑retryable status
		}
		if i < opts.Retries {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return resp, err
}

// Helper utilities ---------------------------------------------------------

func readPossiblyFile(v string) (string, error) {
	if strings.HasPrefix(v, "@") {
		path := strings.TrimPrefix(v, "@")
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read file %s: %w", path, err)
		}
		return string(b), nil
	}
	return v, nil
}

func isIdempotent(m string) bool {
	switch strings.ToUpper(m) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
