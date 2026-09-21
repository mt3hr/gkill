package common

// Android で libc（SQLite の 'localtime'）へ端末のゾーンを教える経路の回帰テスト。
// Android そのものは CI に無いので、packed tzdata の切り出し・TZ 環境変数の組み立て・
// fallback の POSIX 文字列を、合成データで固定する。
// documents/adr/0220-sqlite-localtime-follows-libc-zone.md

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
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

// 未展開の "$VAR/gkill"（既定の --gkill_home_dir はこの形）や相対パスを渡しても、TZ には展開済みの絶対パスが入り、
// CWD に文字どおり "$VAR" という名のディレクトリを掘らないこと。
// musl は TZ=:<相対名> を zoneinfo ディレクトリで探して無ければ黙って UTC にするので、ここが崩れると
// Termux（--gkill_home_dir 無しで起動する）で時間帯検索が0件に戻る（2026-09-16 に実際にそうなった）。
func TestApplyLibcTimezoneForExpandsEnvAndUsesAbsolutePath(t *testing.T) {
	t.Setenv("TZ", "") // 終了時に元へ戻す
	home := t.TempDir()
	t.Setenv("GKILL_TEST_HOME", home)
	t.Chdir(t.TempDir()) // CWD を隔離して、汚したかどうかを見られるようにする
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tokyo := fakeTZif("tokyo")
	loader := func(string) ([]byte, error) { return tokyo, nil }

	applied := applyLibcTimezoneFor("Asia/Tokyo", "$GKILL_TEST_HOME/gkill", loader)
	want := filepath.Join(home, "gkill", "tz", "localtime")
	if applied != "TZ=:"+want {
		t.Fatalf("applied = %q, want %q", applied, "TZ=:"+want)
	}
	got := os.Getenv("TZ")
	if got != ":"+want {
		t.Fatalf("TZ = %q, want %q", got, ":"+want)
	}
	if strings.Contains(got, "$") || !filepath.IsAbs(strings.TrimPrefix(got, ":")) {
		t.Fatalf("TZ が展開済みの絶対パスでない: %q", got)
	}
	if written, err := os.ReadFile(want); err != nil || !bytes.Equal(written, tokyo) {
		t.Fatalf("展開先に TZif が書かれていない: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cwd, "$GKILL_TEST_HOME")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("CWD 直下に文字どおり $GKILL_TEST_HOME というディレクトリを掘った: %v", err)
	}

	// 相対パスでも絶対パスにしてから渡す
	rel := filepath.Join("rel", "gkill")
	applied = applyLibcTimezoneFor("Asia/Tokyo", rel, loader)
	wantRel, err := filepath.Abs(filepath.Join(rel, "tz", "localtime"))
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(wantRel) || applied != "TZ=:"+wantRel {
		t.Fatalf("相対パス: applied = %q, want %q", applied, "TZ=:"+wantRel)
	}
	if got := os.Getenv("TZ"); got != ":"+wantRel {
		t.Fatalf("相対パス: TZ = %q", got)
	}
	if _, err := os.Stat(wantRel); err != nil {
		t.Fatalf("相対パス: TZif が書かれていない: %v", err)
	}

	// 展開すると空になる（未設定の変数だけ）なら TZif 経路を使わず POSIX 文字列へ落ちる
	applied = applyLibcTimezoneFor("Asia/Tokyo", "$GKILL_TEST_UNSET_HOME", loader)
	if !strings.HasPrefix(applied, "TZ=") || strings.HasPrefix(applied, "TZ=:") || !strings.Contains(applied, "expands to empty") {
		t.Fatalf("空に展開されるパスで fallback しない: %q", applied)
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

// InitGkillOptions が libc へ渡すホームは、環境変数を展開した絶対パスであること（933bb66a の回帰点）。
// 未展開の "$HOME/gkill" を渡すと TZ=:$HOME/gkill/tz/localtime のリテラルになり、musl は相対名を
// zoneinfo ディレクトリで探して無ければエラーなしで UTC にする（Termux の既定起動で実際にそうなった）。
func TestInitGkillOptionsPassesExpandedAbsoluteHomeToLibcTimezone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GKILL_TEST_INIT_HOME", home)
	t.Setenv("GKILL_HOME", os.Getenv("GKILL_HOME")) // 終了時に元へ戻す

	originalHomeDir := gkill_options.GkillHomeDir
	originalFn := applyLibcTimezoneFn
	t.Cleanup(func() {
		applyLibcTimezoneFn = originalFn
		gkill_options.GkillHomeDir = originalHomeDir
		InitGkillOptions() // 派生オプション（LibDir 等）を元の値へ戻す
	})

	captured := ""
	applyLibcTimezoneFn = func(gkillHomeDir string) string {
		captured = gkillHomeDir
		return ""
	}
	gkill_options.GkillHomeDir = "$GKILL_TEST_INIT_HOME/gkill"
	InitGkillOptions()

	want := filepath.Clean(filepath.Join(home, "gkill"))
	if captured != want {
		t.Fatalf("applyLibcTimezone に渡したホーム = %q, want %q（展開済み・絶対パス）", captured, want)
	}
	if strings.Contains(captured, "$") || !filepath.IsAbs(captured) {
		t.Fatalf("未展開または相対のまま渡している: %q", captured)
	}
	if got := os.Getenv("GKILL_HOME"); got != want {
		t.Fatalf("GKILL_HOME = %q, want %q（プラグインへ継ぐ値も同じ展開済みパス）", got, want)
	}
}

// loadAndroidTZif は候補パスを順に試し、最初に読めた packed tzdata から切り出す。
// 全部だめなら候補ごとの理由を errors.Join で1つにして返す（どのパスが無かったかが1行で分かる）。
func TestLoadAndroidTZifTriesCandidatesInOrderAndJoinsErrors(t *testing.T) {
	originalPaths := androidPackedTzdataPaths
	t.Cleanup(func() { androidPackedTzdataPaths = originalPaths })

	dir := t.TempDir()
	missing := filepath.Join(dir, "missing", "tzdata")
	older := filepath.Join(dir, "older", "tzdata")
	if err := os.MkdirAll(filepath.Dir(older), 0o755); err != nil {
		t.Fatal(err)
	}
	tokyo := fakeTZif("tokyo-from-older")
	if err := os.WriteFile(older, buildPackedTzdata(t, map[string][]byte{"Asia/Tokyo": tokyo}), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1つ目が無くても2つ目から取れる
	androidPackedTzdataPaths = []string{missing, older}
	got, err := loadAndroidTZif("Asia/Tokyo")
	if err != nil {
		t.Fatalf("2つ目の候補から読めるはず: %v", err)
	}
	if !bytes.Equal(got, tokyo) {
		t.Fatalf("切り出した TZif が違う: %q", got)
	}

	// 読めても名前が無ければ次へ。全部だめなら理由が全部入る
	androidPackedTzdataPaths = []string{missing, older}
	_, err = loadAndroidTZif("Europe/Paris")
	if err == nil {
		t.Fatal("無い名前でエラーにならない")
	}
	for _, want := range []string{missing, older, "Europe/Paris"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("エラーに %q が入っていない: %v", want, err)
		}
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("無いパスの os.ErrNotExist が errors.Join で残っていない: %v", err)
	}
}

// recordingHandler は流れてきたレコードを控える slog.Handler（レベルの絞り込みはしない）。
type recordingHandler struct {
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

func attrsOf(r slog.Record) map[string]string {
	out := map[string]string{}
	r.Attrs(func(a slog.Attr) bool {
		out[a.Key] = a.Value.String()
		return true
	})
	return out
}

// checkSQLiteLocaltime は不一致のとき Error を1行出す（ここが「黙って0件」の唯一の痕跡）。
// 一致なら Debug、問い合わせ自体の失敗は Warn。どれも起動は止めない。
func TestCheckSQLiteLocaltimeLogsErrorWithBothClocksAndHintOnMismatch(t *testing.T) {
	original := checkLocaltimeAgreesWithGo
	t.Cleanup(func() { checkLocaltimeAgreesWithGo = original })
	handler := &recordingHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	checkLocaltimeAgreesWithGo = func(context.Context) (sqlite3impl.LocaltimeAgreement, error) {
		return sqlite3impl.LocaltimeAgreement{GoLocal: "12:09:04", GoWeekday: 3, SQLiteLocal: "03:09:04", SQLiteWeekday: 3}, nil
	}
	checkSQLiteLocaltime(context.Background())
	if len(handler.records) != 1 || handler.records[0].Level != gkill_log.Error {
		t.Fatalf("不一致で Error が1行出るはず: %+v", handler.records)
	}
	r := handler.records[0]
	if !strings.Contains(r.Message, "period-of-time search returns nothing") {
		t.Errorf("症状（時間帯検索が0件）がメッセージに無い: %q", r.Message)
	}
	attrs := attrsOf(r)
	for key, want := range map[string]string{"go_local": `"12:09:04"`, "sqlite_local": `"03:09:04"`} {
		if attrs[key] != want {
			t.Errorf("%s = %q, want %q", key, attrs[key], want)
		}
	}
	if !strings.Contains(attrs["hint"], "TZ=:/absolute/path") {
		t.Errorf("hint に直し方（絶対パスの TZ）が無い: %q", attrs["hint"])
	}

	// 一致なら Debug 1行
	handler.records = nil
	checkLocaltimeAgreesWithGo = func(context.Context) (sqlite3impl.LocaltimeAgreement, error) {
		return sqlite3impl.LocaltimeAgreement{GoLocal: "12:09:04", GoWeekday: 3, SQLiteLocal: "12:09:04", SQLiteWeekday: 3}, nil
	}
	checkSQLiteLocaltime(context.Background())
	if len(handler.records) != 1 || handler.records[0].Level != gkill_log.Debug {
		t.Fatalf("一致なら Debug が1行のはず: %+v", handler.records)
	}

	// 問い合わせが失敗したら Warn 1行（Error にしない: 環境の問題ではなく検査の失敗）
	handler.records = nil
	checkLocaltimeAgreesWithGo = func(context.Context) (sqlite3impl.LocaltimeAgreement, error) {
		return sqlite3impl.LocaltimeAgreement{}, errors.New("no sqlite")
	}
	checkSQLiteLocaltime(context.Background())
	if len(handler.records) != 1 || handler.records[0].Level != gkill_log.Warn {
		t.Fatalf("検査失敗なら Warn が1行のはず: %+v", handler.records)
	}
}
