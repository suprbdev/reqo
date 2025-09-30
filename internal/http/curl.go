package httpx

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

// AsCurl returns a string that reproduces the request with the curl CLI.
func AsCurl(req *http.Request) (string, error) {
	var b strings.Builder
	b.WriteString("curl -X ")
	b.WriteString(req.Method)

	// headers
	for k, vals := range req.Header {
		for _, v := range vals {
			b.WriteString(fmt.Sprintf(` -H "%s: %s"`, escape(k), escape(v)))
		}
	}

	// body (if any)
	if req.Body != nil && req.GetBody != nil {
		bodyCopy, err := req.GetBody()
		if err == nil {
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(bodyCopy)
			data := strings.TrimSpace(buf.String())
			if data != "" {
				b.WriteString(fmt.Sprintf(` --data-raw %s`, shellQuote(data)))
			}
		}
	}

	// URL (including query)
	u := *req.URL
	u.User = nil // omit userinfo for safety
	b.WriteString(" ")
	b.WriteString(shellQuote(u.String()))
	return b.String(), nil
}

// escape makes sure double quotes inside a header value are escaped.
func escape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

// shellQuote puts a string in single‑quotes and escapes existing ones.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	// simple implementation – works for most cases
	s = strings.ReplaceAll(s, `'`, `'\''`)
	return "'" + s + "'"
}
