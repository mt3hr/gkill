package gkill_server_api

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeZip は指定エントリを持つZIPを一時ファイルへ書き、そのパスを返す。
func makeZip(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, data := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return zipPath
}

func TestExtractZip_NormalExtraction(t *testing.T) {
	zipPath := makeZip(t, map[string][]byte{
		"a.txt":     []byte("hello"),
		"sub/b.txt": []byte("world"),
	})
	cacheDir := filepath.Join(t.TempDir(), "cache")

	if err := extractZip(zipPath, cacheDir); err != nil {
		t.Fatalf("normal zip should extract: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(cacheDir, "a.txt"))
	if err != nil || string(got) != "hello" {
		t.Errorf("a.txt not extracted correctly: %q %v", got, err)
	}
	got, err = os.ReadFile(filepath.Join(cacheDir, "sub", "b.txt"))
	if err != nil || string(got) != "world" {
		t.Errorf("sub/b.txt not extracted correctly: %q %v", got, err)
	}
}

func TestExtractZip_RejectsCompressionBomb(t *testing.T) {
	// 2MB の同一バイトは高圧縮比になり、maxZipCompressRatio(200) を超える。
	bomb := bytes.Repeat([]byte("A"), 2*1024*1024)
	zipPath := makeZip(t, map[string][]byte{"bomb.txt": bomb})
	cacheDir := filepath.Join(t.TempDir(), "cache")

	err := extractZip(zipPath, cacheDir)
	if err == nil || !strings.Contains(err.Error(), "zip bomb") {
		t.Fatalf("compression bomb should be rejected, got: %v", err)
	}
	// 中途半端な展開物（.tmp）を残さない。
	if _, statErr := os.Stat(cacheDir + ".tmp"); statErr == nil {
		t.Error("tmp dir should be cleaned up after rejection")
	}
}
