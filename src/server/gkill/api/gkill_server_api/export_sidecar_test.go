package gkill_server_api

import (
	"archive/zip"
	"bytes"
	"regexp"
	"strings"
	"testing"
)

// 同名フォルダの中のファイル名は、元のファイル名をそのまま使わない。
// 使うと Windows の MAX_PATH・使えない文字・ZIP の中の日本語の文字化けに当たり、
// **実ファイル名そのものが個人情報になりうる**。

// safeEntryName は ZIP のエントリ名として安全な字だけを許す。
var safeEntryName = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)

// TestSidecarExtension は、フォルダの中で使う拡張子の決め方を確認する。
func TestSidecarExtension(t *testing.T) {
	for _, testCase := range []struct {
		fileName string
		want     string
	}{
		{"a.jpg", ".jpg"},
		{"a.JPEG", ".jpeg"},
		{"a.mp4", ".mp4"},
		{"拡張子なし", ".bin"},
		{"a.", ".bin"},
		{"a.日本語", ".bin"},
		{"a.verylongextension", ".bin"},
		{"a.tar.gz", ".gz"},
	} {
		t.Run(testCase.fileName, func(t *testing.T) {
			if got := sidecarExtension(testCase.fileName); got != testCase.want {
				t.Errorf("sidecarExtension(%q) = %q, want %q", testCase.fileName, got, testCase.want)
			}
		})
	}
}

// TestExportSidecar_SameContentIsOneEntry は、同じ内容を2回入れても1エントリに
// しかならないことを確認する（内容ハッシュで名前を付けているため）。
func TestExportSidecar_SameContentIsOneEntry(t *testing.T) {
	sidecar := newExportSidecar("出力")
	first := sidecar.add([]byte("同じ中身"), ".jpg")
	second := sidecar.add([]byte("同じ中身"), ".jpg")
	other := sidecar.add([]byte("違う中身"), ".jpg")

	if first != second {
		t.Errorf("同じ内容なのに別の名前になっている: %q と %q", first, second)
	}
	if first == other {
		t.Error("違う内容なのに同じ名前になっている")
	}
	if len(sidecar.files) != 2 {
		t.Errorf("エントリ数 = %d, want 2", len(sidecar.files))
	}
}

// TestBuildExportZip は、ZIP を開き直して中身を確かめる。
func TestBuildExportZip(t *testing.T) {
	sidecar := newExportSidecar("出力テスト")
	// 元のファイル名が日本語でも、フォルダの中の名前は ASCII になる
	imagePath := sidecar.add([]byte("画像のバイト列のつもり"), ".jpg")
	sidecar.add([]byte("動画のバイト列のつもり"), ".mp4")

	zipBytes, err := buildExportZip("出力テスト", "<!doctype html><html></html>", sidecar)
	if err != nil {
		t.Fatalf("buildExportZip failed: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("ZIP として開けない: %v", err)
	}

	names := []string{}
	for _, entry := range reader.File {
		names = append(names, entry.Name)
	}
	if len(names) != 3 {
		t.Fatalf("エントリ数 = %d, want 3（HTML + 2件）: %v", len(names), names)
	}
	if names[0] != "出力テスト.html" {
		t.Errorf("1件目 = %q, want %q", names[0], "出力テスト.html")
	}

	// フォルダの中の名前は ASCII の英数字と `.` `/` だけ
	for _, name := range names[1:] {
		rel, found := strings.CutPrefix(name, "出力テスト/")
		if !found {
			t.Errorf("同名フォルダの外にファイルがある: %q", name)
			continue
		}
		if !safeEntryName.MatchString(rel) {
			t.Errorf("保存できない名前のエントリがある: %q", rel)
		}
	}

	// HTML から参照する相対パスが、実際に ZIP の中にある
	foundReferenced := false
	for _, name := range names {
		if name == imagePath {
			foundReferenced = true
		}
	}
	if !foundReferenced {
		t.Errorf("HTML が参照するパス %q が ZIP に無い", imagePath)
	}
}

// TestBuildExportZip_EntryOrderIsStable は、同じ入力なら同じ並びになることを確認する。
// map の反復順のままだと、同じ内容を出力するたび違うバイト列になる。
func TestBuildExportZip_EntryOrderIsStable(t *testing.T) {
	build := func() []string {
		sidecar := newExportSidecar("出力")
		for _, body := range []string{"a", "b", "c", "d", "e"} {
			sidecar.add([]byte(body), ".jpg")
		}
		zipBytes, err := buildExportZip("出力", "<html></html>", sidecar)
		if err != nil {
			t.Fatalf("buildExportZip failed: %v", err)
		}
		reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
		if err != nil {
			t.Fatalf("ZIP として開けない: %v", err)
		}
		names := []string{}
		for _, entry := range reader.File {
			names = append(names, entry.Name)
		}
		return names
	}

	first := build()
	for range 5 {
		if got := build(); strings.Join(got, "\n") != strings.Join(first, "\n") {
			t.Fatalf("並びが安定していない:\n%v\n%v", first, got)
		}
	}
}
