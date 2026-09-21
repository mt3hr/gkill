package common

// Android のタイムゾーンを Go と libc（SQLite の 'localtime'）の両方へ教える。
//
// Go の time.Local は Android では常に UTC（time/zoneinfo_android.go の initLocal）。
// それは以前から fixTimezone が getprop で直していたが、SQLite の 'localtime' 修飾子は
// Go ではなく libc の localtime_r（modernc = musl 転写）で決まり、musl は
// TZ 環境変数 → /etc/localtime → どちらも無ければ UTC を使う。Android にはどちらも無いので、
// 時間帯フィルタの SQL 段（UTC）と Go 段（JST）が別の壁時計で判定し、9時間より狭い窓の検索が
// エラーも警告も出ないまま常に0件になっていた（2026-09-16）。
//
// TZ=:<パス> のパスは展開済みの絶対パスでなければならない。musl の do_tzset は ':' の後ろが
// '/' でも '.' でも始まらない名前を /usr/share/zoneinfo/ 等の zoneinfo ディレクトリで探し、
// 見つからなければエラーを出さず UTC にする。既定の --gkill_home_dir は "$HOME/gkill" の未展開文字列で、
// 初版はそれをそのまま渡していたので TZ=:$HOME/gkill/tz/localtime（リテラル）になり、
// --gkill_home_dir を渡さない Termux では修正前と同じく UTC のままだった（2026-09-16 同日）。
// documents/adr/0220-sqlite-localtime-follows-libc-zone.md

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// androidZoneName は fixTimezone が getprop から得て time.Local に適用したゾーン名。
// 空なら Go 側は直していない（libc 側も触らない: 両方 UTC のままなら少なくとも揃っている）。
var androidZoneName string

// libcTimezoneApplied は applyLibcTimezone が環境変数に入れた内容（起動ログ用。適用しなければ空）。
var libcTimezoneApplied string

// androidPackedTzdataPaths は Android が持つ packed tzdata の候補。新しい順に試す。
// Android 10 以降は APEX、それ以前は /data/misc（更新済み）→ /system の順で置かれる。
var androidPackedTzdataPaths = []string{
	"/apex/com.android.tzdata/etc/tz/tzdata",
	"/data/misc/zoneinfo/current/tzdata",
	"/system/usr/share/zoneinfo/tzdata",
}

// fixTimezone は Android で Go の time.Local を端末のゾーンに直す。
// init() から呼ばれる（フラグ解析より前。libc 側は GKILL_HOME が要るので applyLibcTimezone で後から）。
func fixTimezone() {
	if runtime.GOOS != "android" {
		return
	}
	out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
	if err != nil {
		return
	}
	name := strings.TrimSpace(string(out))
	z, err := time.LoadLocation(name)
	if err != nil {
		return
	}
	time.Local = z
	androidZoneName = name
}

// applyLibcTimezone は Android で libc（SQLite の 'localtime'）にも fixTimezone と同じゾーンを教える。
//
// 端末の packed tzdata から TZif を取り出して gkillHomeDir/tz/localtime に置き、TZ=:<そのパス> を
// 環境変数に入れる（musl は ':' 始まりの絶対パスを mmap して読む。DST 込みで正確）。
// tzdata が読めなければ POSIX の固定オフセット文字列（例 JST-9）で妥協する。
//
// 必ず最初の SQLite 接続より前に呼ぶこと。modernc の libc は最初の接続を開くときに
// os.Environ() を一度だけ写し取り、以後 os.Setenv しても getenv には映らない。
// 戻り値は起動ログ用の説明（適用しなかったときは空）。
//
// applyLibcTimezoneFn は InitGkillOptions が呼ぶ入口。テストが差し替えて「展開済みの絶対パスが渡ること」を見る。
var applyLibcTimezoneFn = applyLibcTimezone

func applyLibcTimezone(gkillHomeDir string) string {
	if runtime.GOOS != "android" || androidZoneName == "" {
		return ""
	}
	return applyLibcTimezoneFor(androidZoneName, gkillHomeDir, loadAndroidTZif)
}

// applyLibcTimezoneFor は applyLibcTimezone の本体。tzdata の読み手を差し替えられるようにしてある（テスト用）。
//
// gkillHomeDir は環境変数を展開し絶対パスにしてから使う（呼び出し側が展開済みでも二重の展開は無害）。
// 絶対パスにできなければ TZif 経路は使わず POSIX 文字列へ落とす —— 相対名の TZ は musl が黙って UTC にするので、
// 半年ずれるかもしれない fallback のほうがまだ正しい。
func applyLibcTimezoneFor(zoneName string, gkillHomeDir string, loadTZif func(name string) ([]byte, error)) string {
	path, err := localtimeTZifPath(gkillHomeDir)
	if err == nil {
		var tzif []byte
		if tzif, err = loadTZif(zoneName); err == nil {
			if err = writeFileIfChanged(path, tzif); err == nil {
				if err = os.Setenv("TZ", ":"+path); err == nil {
					return "TZ=:" + path
				}
			}
		}
	}
	posix := posixTZString(time.Now().In(time.Local))
	if setErr := os.Setenv("TZ", posix); setErr != nil {
		return ""
	}
	return fmt.Sprintf("TZ=%s (TZif unavailable: %v)", posix, err)
}

// localtimeTZifPath は TZif を置く場所 <gkillHomeDir>/tz/localtime を、環境変数を展開した絶対パスで返す。
// musl は TZ=:<パス> のパスが '/' か '.' で始まらないと zoneinfo ディレクトリの相対名として扱うので、
// 絶対パスにできないときはエラーにする（呼び出し側は POSIX 文字列へ落とす）。
func localtimeTZifPath(gkillHomeDir string) (string, error) {
	expanded := os.ExpandEnv(gkillHomeDir)
	if expanded == "" {
		return "", fmt.Errorf("gkill home dir %q expands to empty", gkillHomeDir)
	}
	path, err := filepath.Abs(filepath.Join(expanded, "tz", "localtime"))
	if err != nil {
		return "", fmt.Errorf("resolve absolute path of %q: %w", gkillHomeDir, err)
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("gkill home dir %q did not resolve to an absolute path: %q", gkillHomeDir, path)
	}
	return path, nil
}

// checkSQLiteLocaltime は SQLite の 'localtime' と Go の time.Local を突き合わせ、食い違っていれば
// gkill_error.log へ両方の値を出す。呼び出し元へは返さない（ここが唯一の記録なので Debug にしない）。
// 起動は止めない: 時間帯フィルタ以外は正しく動くので、止めるより原因を1行残すほうが役に立つ。
// checkLocaltimeAgreesWithGo は SQLite への問い合わせ。テストが差し替えて不一致時の記録を見る。
var checkLocaltimeAgreesWithGo = sqlite3impl.CheckLocaltimeAgreesWithGo

func checkSQLiteLocaltime(ctx context.Context) {
	result, err := checkLocaltimeAgreesWithGo(ctx)
	if err != nil {
		slog.Log(ctx, gkill_log.Warn, "sqlite localtime check failed", "error", fmt.Sprintf("%q", err))
		return
	}
	if result.Agrees() {
		slog.Log(ctx, gkill_log.Debug, "sqlite localtime agrees with go time.Local",
			"local", fmt.Sprintf("%q", time.Local.String()), "applied", fmt.Sprintf("%q", libcTimezoneApplied))
		return
	}
	slog.Log(ctx, gkill_log.Error, "sqlite localtime differs from go time.Local: period-of-time search returns nothing",
		"go_local", fmt.Sprintf("%q", result.GoLocal), "go_weekday", result.GoWeekday,
		"sqlite_local", fmt.Sprintf("%q", result.SQLiteLocal), "sqlite_weekday", result.SQLiteWeekday,
		"local", fmt.Sprintf("%q", time.Local.String()), "tz_env", fmt.Sprintf("%q", os.Getenv("TZ")),
		"applied", fmt.Sprintf("%q", libcTimezoneApplied),
		"hint", "set TZ to a zone the libc can read (TZ=:/absolute/path/to/TZif or a POSIX string such as JST-9) before starting; a relative path such as TZ=:$HOME/... is looked up under /usr/share/zoneinfo and silently falls back to UTC")
}

// loadAndroidTZif は端末の packed tzdata から name の TZif バイト列を取り出す。
func loadAndroidTZif(name string) ([]byte, error) {
	var errs []error
	for _, path := range androidPackedTzdataPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		tzif, err := extractTZifFromPackedTzdata(data, name)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			continue
		}
		return tzif, nil
	}
	return nil, errors.Join(errs...)
}

// packed tzdata の形（AOSP の ZoneInfoDB.java / Go の time/zoneinfo_android.go と同じ）:
//
//	0..12   "tzdata<version>\0"
//	12..16  index_offset (big endian)
//	16..20  data_offset
//	20..24  final_offset
//	index   [name 40 bytes（NUL 詰め）][offset 4][length 4][raw_utc_offset 4] × N
//	data    TZif の並び。offset は data_offset からの相対
const (
	packedTzdataMagic      = "tzdata"
	packedTzdataHeaderSize = 12 + 3*4
	packedTzdataNameSize   = 40
	packedTzdataEntrySize  = packedTzdataNameSize + 3*4
)

// extractTZifFromPackedTzdata は packed tzdata のバイト列から name の TZif を切り出す。
func extractTZifFromPackedTzdata(data []byte, name string) ([]byte, error) {
	if len(name) == 0 || len(name) > packedTzdataNameSize {
		return nil, fmt.Errorf("zone name %q is empty or longer than %d bytes", name, packedTzdataNameSize)
	}
	if len(data) < packedTzdataHeaderSize || !bytes.HasPrefix(data, []byte(packedTzdataMagic)) {
		return nil, errors.New("corrupt packed tzdata (header)")
	}
	indexOff := int(binary.BigEndian.Uint32(data[12:16]))
	dataOff := int(binary.BigEndian.Uint32(data[16:20]))
	if indexOff < packedTzdataHeaderSize || dataOff < indexOff || dataOff > len(data) {
		return nil, errors.New("corrupt packed tzdata (offsets)")
	}
	for pos := indexOff; pos+packedTzdataEntrySize <= dataOff; pos += packedTzdataEntrySize {
		entry := data[pos : pos+packedTzdataEntrySize]
		entryName := string(bytes.TrimRight(entry[:packedTzdataNameSize], "\x00"))
		if entryName != name {
			continue
		}
		off := int(binary.BigEndian.Uint32(entry[packedTzdataNameSize : packedTzdataNameSize+4]))
		size := int(binary.BigEndian.Uint32(entry[packedTzdataNameSize+4 : packedTzdataNameSize+8]))
		start := dataOff + off
		if off < 0 || size <= 0 || start > len(data) || size > len(data)-start {
			return nil, fmt.Errorf("corrupt packed tzdata (entry %q out of range)", name)
		}
		tzif := data[start : start+size]
		if !bytes.HasPrefix(tzif, []byte("TZif")) {
			return nil, fmt.Errorf("entry %q is not TZif", name)
		}
		return tzif, nil
	}
	return nil, fmt.Errorf("zone %q not found in packed tzdata", name)
}

// posixTZString は t の時点のゾーン略称とオフセットから POSIX TZ 文字列を作る（例 "JST-9"、"<+0530>-5:30"）。
//
// POSIX の符号は UTC から「西へ」正なので、UTC+9 は -9。略称が英字3文字以上でなければ <> で囲む
// （musl は <...> 形式を受理する）。夏時間の規則は載らないので、DST のある地域では
// 半年ずれる。あくまで tzdata が読めなかったときの fallback。
func posixTZString(t time.Time) string {
	abbr, offset := t.Zone()
	if !isPosixAlphaAbbr(abbr) {
		abbr = "<" + abbr + ">"
	}
	sign := "-"
	switch {
	case offset == 0:
		sign = ""
	case offset < 0:
		sign = "+"
		offset = -offset
	}
	hours, minutes := offset/3600, (offset%3600)/60
	if minutes == 0 {
		return fmt.Sprintf("%s%s%d", abbr, sign, hours)
	}
	return fmt.Sprintf("%s%s%d:%02d", abbr, sign, hours, minutes)
}

// isPosixAlphaAbbr は POSIX の std 名として素のまま書ける略称（英字3文字以上）か。
func isPosixAlphaAbbr(abbr string) bool {
	if len(abbr) < 3 {
		return false
	}
	for _, r := range abbr {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

// writeFileIfChanged は中身が同じなら書き直さず、違えば tmp + rename で原子的に置く。
func writeFileIfChanged(path string, content []byte) error {
	if current, err := os.ReadFile(path); err == nil && bytes.Equal(current, content) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".localtime-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
