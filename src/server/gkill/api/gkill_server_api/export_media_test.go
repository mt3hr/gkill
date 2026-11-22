package gkill_server_api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// 出力 HTML のメディアは「鍵が画面の引き方と一字一句同じ」でないと引けない。
// 外れても例外は出ず、**出力 HTML がその画像を取りに通信する**（通信しない約束が破れる）。

// TestExportThumbQuery_MatchesClient は、サムネイルの鍵がクライアントの
// build_media_url と同じ形であることを確認する。
func TestExportThumbQuery_MatchesClient(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "client", "classes", "use-idf-kyou-view.ts"))
	if err != nil {
		t.Fatalf("クライアントのソースが読めない: %v", err)
	}
	client := string(source)

	for _, want := range []string{
		`"?` + exportThumbQuery + `"`,
		`"?` + exportVideoThumbQuery + `"`,
	} {
		if !strings.Contains(client, want) {
			t.Errorf("クライアントの build_media_url に %s が無い（鍵が食い違うと出力 HTML が通信する）", want)
		}
	}
}

// TestSplitIDFFileURL は /files/{rep}/{rel} の分け方を確認する。
func TestSplitIDFFileURL(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		fileURL string
		rep     string
		rel     string
		ok      bool
	}{
		{"普通の形", "/files/RepA/dir/a.jpg", "RepA", "dir/a.jpg", true},
		{"エスケープはそのまま渡す", "/files/RepA/%E3%81%82.jpg", "RepA", "%E3%81%82.jpg", true},
		{"rep だけ", "/files/RepA/", "", "", false},
		{"rel だけ", "/files/a.jpg", "", "", false},
		{"別の接頭辞", "/api/get_kyous", "", "", false},
		{"空", "", "", "", false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rep, rel, ok := splitIDFFileURL(testCase.fileURL)
			if ok != testCase.ok || rep != testCase.rep || rel != testCase.rel {
				t.Errorf("splitIDFFileURL(%q) = (%q, %q, %v), want (%q, %q, %v)",
					testCase.fileURL, rep, rel, ok, testCase.rep, testCase.rel, testCase.ok)
			}
		})
	}
}

// TestIsExportTextFile は、本文を焼き込む対象の判定がクライアントと同じであることを確認する。
// ずれると「画面には出るのに本文だけ空」になる。
func TestIsExportTextFile(t *testing.T) {
	for _, testCase := range []struct {
		name string
		idf  reps.IDFKyou
		want bool
	}{
		{"txt", reps.IDFKyou{TargetFile: "memo.txt"}, true},
		{"md", reps.IDFKyou{TargetFile: "memo.md"}, true},
		{"markdown", reps.IDFKyou{TargetFile: "memo.markdown"}, true},
		{"大文字でも同じ", reps.IDFKyou{TargetFile: "MEMO.TXT"}, true},
		{"画像は対象外", reps.IDFKyou{TargetFile: "a.txt", IsImage: true}, false},
		{"動画は対象外", reps.IDFKyou{TargetFile: "a.txt", IsVideo: true}, false},
		{"音声は対象外", reps.IDFKyou{TargetFile: "a.txt", IsAudio: true}, false},
		{"zip は対象外", reps.IDFKyou{TargetFile: "a.txt", IsZip: true}, false},
		{"別の拡張子", reps.IDFKyou{TargetFile: "a.pdf"}, false},
		{"拡張子なし", reps.IDFKyou{TargetFile: "README"}, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isExportTextFile(testCase.idf); got != testCase.want {
				t.Errorf("isExportTextFile(%+v) = %v, want %v", testCase.idf, got, testCase.want)
			}
		})
	}
}

// TestExportMediaContentType は data: URI に書く種別を確認する。
// 引けないものを image/jpeg などに決め打ちすると、別の種別のファイルが壊れて表示される。
func TestExportMediaContentType(t *testing.T) {
	for _, testCase := range []struct {
		fileName string
		want     string
	}{
		{"a.jpg", "image/jpeg"},
		{"a.JPEG", "image/jpeg"},
		{"a.png", "image/png"},
		{"a.mp4", "video/mp4"},
		{"a.unknown", "application/octet-stream"},
		{"a", "application/octet-stream"},
	} {
		t.Run(testCase.fileName, func(t *testing.T) {
			got := exportMediaContentType(reps.IDFKyou{TargetFile: testCase.fileName})
			if got != testCase.want {
				t.Errorf("exportMediaContentType(%q) = %q, want %q", testCase.fileName, got, testCase.want)
			}
		})
	}
}
