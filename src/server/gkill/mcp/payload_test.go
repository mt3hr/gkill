package mcp

import (
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// StripFilePaths は応答から file_path を再帰的に落とす（server_base.go の唯一の実装。
// リモートクライアントにこのマシンの絶対パスを渡さないための出口）。
func TestStripFilePathsRemovesEveryNestedFilePath(t *testing.T) {
	t.Run("removes file_path at every depth and keeps the other keys", func(t *testing.T) {
		inner := jsonobj.Obj("file_path", "/home/x/b.jpg", "file_name", "b.jpg")
		payload := jsonobj.Obj(
			"file_path", "/home/x/a.jpg",
			"file_name", "a.jpg",
			"nested", inner,
			"list", []any{jsonobj.Obj("file_path", "/home/x/c.jpg", "id", "c"), "plain string", 3},
		)
		got := StripFilePaths(payload).(*jsonobj.Object)
		_, has := got.String("file_path")
		expectTrue(t, !has, "top-level file_path survived")
		name, _ := got.String("file_name")
		expectEqual(t, name, "a.jpg")
		nested, _ := got.Object("nested")
		_, has = nested.String("file_path")
		expectTrue(t, !has, "nested file_path survived")
		list, _ := jsonobj.AsArray(got.Value("list"))
		_, has = list[0].(*jsonobj.Object).String("file_path")
		expectTrue(t, !has, "file_path inside an array survived")
		expectEqual(t, list[1], "plain string")
	})

	t.Run("non-object values pass through unchanged", func(t *testing.T) {
		expectEqual(t, StripFilePaths("s"), "s")
		expectEqual(t, StripFilePaths(nil), nil)
		var nilObject *jsonobj.Object
		expectTrue(t, StripFilePaths(nilObject) == nilObject, "nil object changed")
	})
}

// NormalizeMimeType は Content-Type からパラメータ（charset 等）と前後の空白を落とした media type だけを返す。
// http_transport.go の配信 MIME に使う。
func TestNormalizeMimeType(t *testing.T) {
	cases := map[any]string{
		"image/jpeg":                     "image/jpeg",
		" text/html; charset=utf-8 ":     "text/html",
		"application/json;charset=UTF-8": "application/json",
		"":                               "",
		nil:                              "",
	}
	for in, want := range cases {
		expectEqual(t, NormalizeMimeType(in), want)
	}
	expectEqual(t, NormalizeMimeType(jsonobj.Undefined), "")
}
