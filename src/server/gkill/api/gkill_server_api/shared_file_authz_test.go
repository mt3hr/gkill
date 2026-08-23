package gkill_server_api

import (
	"runtime"
	"testing"
)

// C-03 の核心: 共有経路のファイル配信は「共有クエリの結果に含まれるファイル」だけを
// 許可し、同一rep内の兄弟ファイルは 403 にする。許可集合の突き合わせと、URLパスの
// 正規化（buildIDFFileURL 側の cleanRelativeURLPath と同一）を固定する。
func TestIsSharedFileAllowed(t *testing.T) {
	allowed := map[string]map[string]struct{}{
		"Files": {
			"shared.png":     {},
			"sub/nested.png": {},
		},
	}

	cases := []struct {
		name    string
		repName string
		relPath string
		want    bool
	}{
		{"共有対象ファイルは許可", "Files", "shared.png", true},
		{"共有結果に無い兄弟ファイルは拒否（C-03の本丸）", "Files", "secret.png", false},
		{"サブディレクトリの共有ファイルは許可", "Files", "sub/nested.png", true},
		{"別repは拒否", "OtherRep", "shared.png", false},
		{"表記揺れ（先頭スラッシュ）は畳んで許可", "Files", "/shared.png", true},
		{"表記揺れ（連続スラッシュ）は畳んで許可", "Files", "sub//nested.png", true},
		{"traversal を含む要求は正規化されて拒否", "Files", "sub/../secret.png", false},
		{"空パスは拒否", "Files", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := isSharedFileAllowed(allowed, c.repName, c.relPath)
			if got != c.want {
				t.Errorf("isSharedFileAllowed(%q, %q) = %v, want %v", c.repName, c.relPath, got, c.want)
			}
		})
	}
}

// Mi ビュー等で許可集合が空のときは、どのファイルも 403 になる。
func TestIsSharedFileAllowed_EmptyAllowSetRejectsEverything(t *testing.T) {
	allowed := map[string]map[string]struct{}{}
	if isSharedFileAllowed(allowed, "Files", "anything.png") {
		t.Error("empty allow set must reject every file")
	}
}

// cleanSharedRelPath は dao/reps の cleanRelativeURLPath と同じ正規化であること。
func TestCleanSharedRelPath(t *testing.T) {
	cases := map[string]string{
		"a.png":            "a.png",
		"/a.png":           "a.png",
		"sub//b.png":       "sub/b.png",
		"sub/./b.png":      "sub/b.png",
		"sub/../b.png":     "b.png",
		"":                 "",
		".":                "",
		"../../etc/passwd": "etc/passwd",
	}
	// バックスラッシュの畳み方は filepath.ToSlash 経由で OS のパス意味論に従う。
	// Windows: `\` は区切り文字。畳まないと同一ファイルの認可キーが分かれるので畳む。
	// Unix:    `\` はファイル名の正当な1文字。畳むと `a\b.png`（1ファイル）と
	//          `a/b.png`（dir a 内の b.png）という別ファイルが同じ認可キーへ衝突し、
	//          片方の共有でもう片方が配信されうるため、畳まないのが安全側。
	// 期待値を無条件に "a/b.png" にすると Linux の CI でだけ落ちる（ToSlash は no-op）。
	if runtime.GOOS == "windows" {
		cases["a\\b.png"] = "a/b.png"
	} else {
		cases["a\\b.png"] = "a\\b.png"
	}
	for in, want := range cases {
		if got := cleanSharedRelPath(in); got != want {
			t.Errorf("cleanSharedRelPath(%q) = %q, want %q", in, got, want)
		}
	}
}
