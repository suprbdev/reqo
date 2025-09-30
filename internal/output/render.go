package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/itchyny/gojq"
)

// RenderOpts controls how the response is printed.
type RenderOpts struct {
	ShowHeaders bool // -i / --include
	RawOutput   bool // --raw
	JQExpr      string
}

// Render writes the HTTP response to out according to opts.
func Render(resp *http.Response, out io.Writer, opts RenderOpts) error {
	if opts.ShowHeaders {
		fmt.Fprintf(out, "HTTP/%.1f %d %s\n", float64(resp.ProtoMajor)+float64(resp.ProtoMinor)/10, resp.StatusCode, resp.Status)
		for k, vv := range resp.Header {
			for _, v := range vv {
				fmt.Fprintf(out, "%s: %s\n", k, v)
			}
		}
		fmt.Fprintln(out)
	}

	if opts.RawOutput {
		_, err := io.Copy(out, resp.Body)
		return err
	}

	// Read all body (small responses – for large streams you would stream).
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var pretty bytes.Buffer
		if err = json.Indent(&pretty, bodyBytes, "", "  "); err == nil {
			bodyBytes = pretty.Bytes()
		}
	}

	// JQ filter if requested.
	if opts.JQExpr != "" {
		query, err := gojq.Parse(opts.JQExpr)
		if err != nil {
			return fmt.Errorf("invalid jq expression: %w", err)
		}
		var data interface{}
		if err = json.Unmarshal(bodyBytes, &data); err != nil {
			return fmt.Errorf("cannot unmarshal JSON for jq: %w", err)
		}
		iter := query.Run(data)
		results := []interface{}{}
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, isErr := v.(error); isErr {
				return fmt.Errorf("jq execution error: %w", err)
			}
			results = append(results, v)
		}
		outBytes, _ := json.MarshalIndent(results, "", "  ")
		bodyBytes = outBytes
	}

	_, err = out.Write(bodyBytes)
	if len(bodyBytes) > 0 && bodyBytes[len(bodyBytes)-1] != '\n' {
		fmt.Fprintln(out)
	}
	return err
}
