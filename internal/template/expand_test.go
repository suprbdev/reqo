package template

import (
	"os"
	"testing"
)

func TestExpand_FromVars(t *testing.T) {
	vars := map[string]string{"name": "world"}
	got := Expand("hello ${name}", vars)
	want := "hello world"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_MultipleVars(t *testing.T) {
	vars := map[string]string{"first": "John", "last": "Doe"}
	got := Expand("${first} ${last}", vars)
	want := "John Doe"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_NoVars(t *testing.T) {
	got := Expand("plain text", nil)
	want := "plain text"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_VarInURL(t *testing.T) {
	vars := map[string]string{"id": "123"}
	got := Expand("/users/${id}/posts", vars)
	want := "/users/123/posts"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_MissingVar(t *testing.T) {
	got := Expand("hello ${missing}", nil)
	want := "hello ${missing}"
	if got != want {
		t.Errorf("Expand() = %q, want %q (placeholder should remain)", got, want)
	}
}

func TestExpand_EmptyVarName(t *testing.T) {
	got := Expand("hello ${}", nil)
	// ${} matches, key is empty string, not in vars or env -> left as-is
	want := "hello ${}"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_FromEnvVar(t *testing.T) {
	t.Setenv("MY_TEST_VAR", "envval")
	got := Expand("val=${MY_TEST_VAR}", nil)
	want := "val=envval"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_ReqoPrefixedEnv_Preferred(t *testing.T) {
	t.Setenv("REQO_TOKEN", "secret")
	t.Setenv("TOKEN", "plain")
	got := Expand("Bearer ${TOKEN}", nil)
	want := "Bearer secret"
	if got != want {
		t.Errorf("Expand() = %q, want %q (REQO_ prefix should take priority)", got, want)
	}
}

func TestExpand_FallbackToPlainEnv(t *testing.T) {
	os.Unsetenv("REQO_USERTOKEN")
	t.Setenv("USERTOKEN", "plain")
	got := Expand("Bearer ${USERTOKEN}", nil)
	want := "Bearer plain"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_VarOverridesEnv(t *testing.T) {
	t.Setenv("REQO_MODE", "env")
	vars := map[string]string{"MODE": "cli"}
	got := Expand("${MODE}", vars)
	want := "cli"
	if got != want {
		t.Errorf("Expand() = %q, want %q (vars should override env)", got, want)
	}
}

func TestExpand_AdjacentVars(t *testing.T) {
	vars := map[string]string{"a": "1", "b": "2"}
	got := Expand("${a}${b}", vars)
	want := "12"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_VarInJSON(t *testing.T) {
	vars := map[string]string{"name": "John", "email": "john@example.com"}
	input := `{"name": "${name}", "email": "${email}"}`
	want := `{"name": "John", "email": "john@example.com"}`
	got := Expand(input, vars)
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_VarWithSpecialChars(t *testing.T) {
	vars := map[string]string{"path": "/a/b/c?q=1&z=2"}
	got := Expand("https://example.com${path}", vars)
	want := "https://example.com/a/b/c?q=1&z=2"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_NoPlaceholder(t *testing.T) {
	got := Expand("just text", map[string]string{"x": "y"})
	want := "just text"
	if got != want {
		t.Errorf("Expand() = %q, want %q", got, want)
	}
}

func TestExpand_EmptyString(t *testing.T) {
	got := Expand("", map[string]string{"x": "y"})
	if got != "" {
		t.Errorf("Expand(\"\") = %q, want empty", got)
	}
}

func TestExpandMap(t *testing.T) {
	in := map[string]string{
		"name":  "user_${id}",
		"email": "${user}@example.com",
	}
	vars := map[string]string{
		"id":   "42",
		"user": "john",
	}
	out := ExpandMap(in, vars)
	if out["name"] != "user_42" {
		t.Errorf("name = %q, want %q", out["name"], "user_42")
	}
	if out["email"] != "john@example.com" {
		t.Errorf("email = %q, want %q", out["email"], "john@example.com")
	}
}

func TestExpandMap_Empty(t *testing.T) {
	out := ExpandMap(map[string]string{}, nil)
	if len(out) != 0 {
		t.Errorf("expected empty map")
	}
}

func TestExpandMap_DoesNotMutateInput(t *testing.T) {
	in := map[string]string{"k": "${v}"}
	_ = ExpandMap(in, map[string]string{"v": "expanded"})
	if in["k"] != "${v}" {
		t.Errorf("input map was mutated")
	}
}
