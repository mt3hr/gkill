package mcp

// テスト共通のヘルパ。vitest の expect(...).toEqual / toThrow(GkillApiError) / toThrow(/re/) に相当する。

import (
	"regexp"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

func obj(kv ...any) *jsonobj.Object { return jsonobj.Obj(kv...) }

func arr(items ...any) []any { return jsonobj.Arr(items...) }

func strs(items ...string) []any { return jsonobj.Strings(items...) }

func parse(t testing.TB, src string) any {
	t.Helper()
	v, err := jsonobj.Unmarshal([]byte(src))
	if err != nil {
		t.Fatalf("invalid JSON in test: %v\n%s", err, src)
	}
	return v
}

func parseObj(t testing.TB, src string) *jsonobj.Object {
	t.Helper()
	o, ok := parse(t, src).(*jsonobj.Object)
	if !ok {
		t.Fatalf("not an object: %s", src)
	}
	return o
}

// expectEqual は toEqual。
func expectEqual(t testing.TB, got, want any) {
	t.Helper()
	if !jsonobj.Equal(got, want) {
		t.Fatalf("not equal\n got: %s\nwant: %s", jsonobj.MarshalString(got), jsonobj.MarshalString(want))
	}
}

// expectNotEqual は not.toEqual。
func expectNotEqual(t testing.TB, got, want any) {
	t.Helper()
	if jsonobj.Equal(got, want) {
		t.Fatalf("unexpectedly equal: %s", jsonobj.MarshalString(got))
	}
}

// expectContains は toMatchObject。
func expectContains(t testing.TB, got, subset any) {
	t.Helper()
	if !jsonobj.Contains(got, subset) {
		t.Fatalf("does not contain\n got: %s\nsubset: %s", jsonobj.MarshalString(got), jsonobj.MarshalString(subset))
	}
}

// expectNoError は not.toThrow。
func expectNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// expectGkillApiError は toThrow(GkillApiError)。
func expectGkillApiError(t testing.TB, err error) *GkillApiError {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a GkillApiError, got nil")
	}
	apiErr, ok := AsGkillApiError(err)
	if !ok {
		t.Fatalf("expected a GkillApiError, got %T: %v", err, err)
	}
	return apiErr
}

// expectErrorMatches は toThrow(/pattern/)。
func expectErrorMatches(t testing.TB, err error, pattern string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error matching %q, got nil", pattern)
	}
	if !regexp.MustCompile(pattern).MatchString(err.Error()) {
		t.Fatalf("error %q does not match %q", err.Error(), pattern)
	}
}

// expectErrorContains は toThrow("substring")。
func expectErrorContains(t testing.TB, err error, substring string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", substring)
	}
	if !containsString(err.Error(), substring) {
		t.Fatalf("error %q does not contain %q", err.Error(), substring)
	}
}

func containsString(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func expectTrue(t testing.TB, cond bool, format string, args ...any) {
	t.Helper()
	if !cond {
		t.Fatalf(format, args...)
	}
}

func mustMatch(t testing.TB, s, pattern string) {
	t.Helper()
	if !regexp.MustCompile(pattern).MatchString(s) {
		t.Fatalf("%q does not match %q", s, pattern)
	}
}

func mustNotMatch(t testing.TB, s, pattern string) {
	t.Helper()
	if regexp.MustCompile(pattern).MatchString(s) {
		t.Fatalf("%q unexpectedly matches %q", s, pattern)
	}
}

func mustContain(t testing.TB, s, sub string) {
	t.Helper()
	if !containsString(s, sub) {
		t.Fatalf("%q does not contain %q", s, sub)
	}
}

func mustNotContain(t testing.TB, s, sub string) {
	t.Helper()
	if containsString(s, sub) {
		t.Fatalf("%q unexpectedly contains %q", s, sub)
	}
}

func str(t testing.TB, v any) string {
	t.Helper()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("not a string: %T (%v)", v, v)
	}
	return s
}

func strAt(t testing.TB, o *jsonobj.Object, key string) string {
	t.Helper()
	return str(t, o.Value(key))
}

func objAt(t testing.TB, v any, keys ...string) *jsonobj.Object {
	t.Helper()
	cur := v
	if len(keys) > 0 {
		o, ok := v.(*jsonobj.Object)
		if !ok {
			t.Fatalf("not an object: %T", v)
		}
		cur = o.Path(keys...)
	}
	o, ok := cur.(*jsonobj.Object)
	if !ok || o == nil {
		t.Fatalf("path %v is not an object: %T", keys, cur)
	}
	return o
}

func arrAt(t testing.TB, v any, keys ...string) []any {
	t.Helper()
	cur := v
	if len(keys) > 0 {
		o, ok := v.(*jsonobj.Object)
		if !ok {
			t.Fatalf("not an object: %T", v)
		}
		cur = o.Path(keys...)
	}
	a, ok := jsonobj.AsArray(cur)
	if !ok {
		t.Fatalf("path %v is not an array: %T", keys, cur)
	}
	return a
}

func floatOf(t testing.TB, v any) float64 {
	t.Helper()
	f, ok := jsonobj.ToFloat(v)
	if !ok {
		t.Fatalf("not a number: %T (%v)", v, v)
	}
	return f
}

func toolNames(tools []*jsonobj.Object) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		n, _ := tool.String("name")
		out = append(out, n)
	}
	return out
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
