package common

// Android で libc（SQLite の 'localtime'）へ端末のゾーンを教える経路の回帰テスト。
// Android そのものは CI に無いので、packed tzdata の切り出し・TZ 環境変数の組み立て・
// fallback の POSIX 文字列を、合成データで固定する。
// documents/adr/0220-sqlite-localtime-follows-libc-zone.md

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildPackedTzdata は AOSP の packed tzdata と同じ形（ヘッダ 24 バイト・40 バイト名の索引・データ）を作る。
func buildPackedTzdata(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	// 索引は名前順（実物も辞書順で並ぶ）
	for i := range names {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	indexOff := packedTzdataHeaderSize
	dataOff := indexOff + packedTzdataEntrySize*len(names)

	var index, body bytes.Buffer
	for _, name := range names {
		nameField := make([]byte, packedTzdataNameSize)
		copy(nameField, name)
		index.Write(nameField)
		_ = binary.Write(&index, binary.BigEndian, uint32(body.Len()))
		_ = binary.Write(&index, binary.BigEndian, uint32(len(entries[name])))
		_ = binary.Write(&index, binary.BigEndian, uint32(0)) // raw_utc_offset は使わない
		body.Write(entries[name])
	}

	out := []byte("tzdata2025a\x00")
	out = binary.BigEndian.AppendUint32(out, uint32(indexOff))
	out = binary.BigEndian.AppendUint32(out, uint32(dataOff))
	out = binary.BigEndian.AppendUint32(out, uint32(dataOff+body.Len())) // final_offset
	out = append(out, index.Bytes()...)
	out = append(out, body.Bytes()...)
	return out
}

func fakeTZif(tag string) []byte {
	return append([]byte("TZif2\x00\x00\x00"), []byte(tag)...)
}

func TestExtractTZifFromPackedTzdata(t *testing.T) {
	tokyo := fakeTZif("tokyo")
	packed := buildPackedTzdata(t, map[string][]byte{
		"America/New_York": fakeTZif("new_york"),
		"Asia/Tokyo":       tokyo,
		"UTC":              fakeTZif("utc"),
	})

	got, err := extractTZifFromPackedTzdata(packed, "Asia/Tokyo")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !bytes.Equal(got, tokyo) {
		t.Fatalf("取り出した TZif が違う: %q", got)
	}

	if _, err := extractTZifFromPackedTzdata(packed, "Asia/Nowhere"); err == nil {
		t.Fatal("無いゾーンでエラーにならない")
	}
	// 前方一致で別のゾーンを返してはいけない（"Asia/Tok" で Asia/Tokyo が出る等）
	if _, err := extractTZifFromPackedTzdata(packed, "Asia/Tok"); err == nil {
		t.Fatal("名前の前方一致で別のゾーンを返した")
	}
	if _, err := extractTZifFromPackedTzdata([]byte("not tzdata at all"), "Asia/Tokyo"); err == nil {
		t.Fatal("壊れたヘッダでエラーにならない")
	}
	// 索引が範囲外を指す
	broken := append([]byte{}, packed...)
	binary.BigEndian.PutUint32(broken[16:20], uint32(len(broken)+1))
	if _, err := extractTZifFromPackedTzdata(broken, "Asia/Tokyo"); err == nil {
		t.Fatal("data_offset が範囲外でもエラーにならない")
	}
	// エントリの中身が TZif でない
	notTZif := buildPackedTzdata(t, map[string][]byte{"Asia/Tokyo": []byte("garbage")})
	if _, err := extractTZifFromPackedTzdata(notTZif, "Asia/Tokyo"); err == nil {
		t.Fatal("TZif でない中身を返した")
	}
}

func TestPosixTZString(t *testing.T) {
	cases := []struct {
		zone string
		want string
	}{
		{"Asia/Tokyo", "JST-9"},
		{"Asia/Kolkata", "IST-5:30"},
		{"UTC", "UTC0"},
		{"Etc/GMT+5", "<-05>+5"}, // 略称が英字でないので <> で囲む
	}
	// 夏時間の無いゾーンだけ選んでいるので、時点はいつでも同じ答えになる
	at := time.Date(2026, 9, 16, 12, 9, 4, 0, time.UTC)
	for _, c := range cases {
		z, err := time.LoadLocation(c.zone)
		if err != nil {
			t.Fatalf("LoadLocation(%s): %v", c.zone, err)
		}
		if got := posixTZString(at.In(z)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.zone, got, c.want)
		}
	}
}

// TZif が取れたときは GKILL_HOME/tz/localtime に置いて TZ=:<パス>、取れなければ POSIX 文字列へ落ちる。
func TestApplyLibcTimezoneFor(t *testing.T) {
	t.Setenv("TZ", "") // 終了時に元へ戻す

	home := t.TempDir()
	tokyo := fakeTZif("tokyo")
	loader := func(name string) ([]byte, error) {
		if name != "Asia/Tokyo" {
			return nil, errors.New("unexpected zone " + name)
		}
		return tokyo, nil
	}

	applied := applyLibcTimezoneFor("Asia/Tokyo", home, loader)
	wantPath := filepath.Join(home, "tz", "localtime")
	if applied != "TZ=:"+wantPath {
		t.Fatalf("applied = %q", applied)
	}
	if got := os.Getenv("TZ"); got != ":"+wantPath {
		t.Fatalf("TZ = %q", got)
	}
	written, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read written TZif: %v", err)
	}
	if !bytes.Equal(written, tokyo) {
		t.Fatalf("書かれた TZif が違う: %q", written)
	}

	// 同じ中身なら書き直さない（mtime が動かない）
	before, _ := os.Stat(wantPath)
	time.Sleep(20 * time.Millisecond)
	_ = applyLibcTimezoneFor("Asia/Tokyo", home, loader)
	after, _ := os.Stat(wantPath)
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("同じ中身なのに書き直した")
	}

	// tzdata が読めない端末では POSIX 固定オフセットへ落ちる
	failing := func(string) ([]byte, error) { return nil, errors.New("no tzdata") }
	applied = applyLibcTimezoneFor("Asia/Tokyo", home, failing)
	if !strings.HasPrefix(applied, "TZ=") || strings.HasPrefix(applied, "TZ=:") {
		t.Fatalf("fallback が POSIX 文字列になっていない: %q", applied)
	}
	if !strings.Contains(applied, "no tzdata") {
		t.Fatalf("fallback の理由が載っていない: %q", applied)
	}
	if got := os.Getenv("TZ"); got == "" || strings.HasPrefix(got, ":") {
		t.Fatalf("fallback の TZ = %q", got)
	}
}

// Android 以外では何もしない（環境変数 TZ を触らない）。
func TestApplyLibcTimezoneIsAndroidOnly(t *testing.T) {
	t.Setenv("TZ", "keep-me")
	if applied := applyLibcTimezone(t.TempDir()); applied != "" {
		t.Fatalf("Android 以外で適用した: %q", applied)
	}
	if got := os.Getenv("TZ"); got != "keep-me" {
		t.Fatalf("TZ を書き換えた: %q", got)
	}
}
