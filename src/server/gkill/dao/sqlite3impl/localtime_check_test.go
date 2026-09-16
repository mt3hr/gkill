package sqlite3impl

// SQLite の 'localtime' と Go の time.Local が食い違う環境（Android など）を検出する
// 自己検査の回帰テスト。
// documents/adr/0220-sqlite-localtime-follows-libc-zone.md

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// envLocaltimeChild が立っていると、テストバイナリは「Go 側だけ Asia/Tokyo に固定して
// 自己検査の結果を印字する子」として動く。TZ 環境変数は親が付け替えて渡す。
// Android の状態（fixTimezone が Go 側だけ直し、libc は TZ / /etc/localtime しか見ない）を
// Linux 上で再現するための仕掛け。
const envLocaltimeChild = "GKILL_TEST_LOCALTIME_CHILD"

func TestMain(m *testing.M) {
	if os.Getenv(envLocaltimeChild) == "1" {
		runLocaltimeChild()
		return
	}
	os.Exit(m.Run())
}

func runLocaltimeChild() {
	z, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	time.Local = z
	result, err := CheckLocaltimeAgreesWithGo(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Printf("agrees=%t go=%s/%d sqlite=%s/%d\n", result.Agrees(), result.GoLocal, result.GoWeekday, result.SQLiteLocal, result.SQLiteWeekday)
}

// runLocaltimeChildWithTZ は TZ だけ差し替えた子プロセスを起動し、stdout の1行を返す。
func runLocaltimeChildWithTZ(t *testing.T, tz string) string {
	t.Helper()
	return runLocaltimeChildWithTZInDir(t, tz, "")
}

// runLocaltimeChildWithTZInDir は runLocaltimeChildWithTZ の作業ディレクトリ指定版（空なら親と同じ）。
// TZ=:<相対名> が CWD のファイルを見るかどうかを確かめるのに使う。
func runLocaltimeChildWithTZInDir(t *testing.T, tz string, dir string) string {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Dir = dir
	env := []string{envLocaltimeChild + "=1"}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "TZ=") || strings.HasPrefix(kv, envLocaltimeChild+"=") {
			continue
		}
		env = append(env, kv)
	}
	if tz != "" {
		env = append(env, "TZ="+tz)
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child(TZ=%q): %v\n%s", tz, err, out)
	}
	return strings.TrimSpace(string(out))
}

// このテストを走らせている環境そのものが揃っていること。
// CI（Linux + tzdata）・Windows・macOS の開発機ではここが通る。
// 落ちたら、その環境で時間帯フィルタが黙って0件になっている。
func TestCheckLocaltimeAgreesWithGo_HostAgrees(t *testing.T) {
	result, err := CheckLocaltimeAgreesWithGo(context.Background())
	if err != nil {
		t.Fatalf("CheckLocaltimeAgreesWithGo: %v", err)
	}
	if !result.Agrees() {
		t.Fatalf("この環境では SQLite の 'localtime' と Go の time.Local が食い違っている: go=%s/%d sqlite=%s/%d",
			result.GoLocal, result.GoWeekday, result.SQLiteLocal, result.SQLiteWeekday)
	}
	// 検査の瞬間は日付境界を含む固定値（UTC 03:09:04 / +09:00 12:09:04）であること。
	// 実行時刻に依存させると、ずれが 24 時間の倍数になる瞬間にすり抜ける
	if got := result.Probe.UTC().Format("2006-01-02T15:04:05Z"); got != "2026-09-16T03:09:04Z" {
		t.Fatalf("probe = %s", got)
	}
}

// Android の状態を Linux で再現する: Go 側は Asia/Tokyo、libc 側はゾーンファイルが見つからず UTC。
// 自己検査がこれを「不一致」と言えなければ、起動ログに何も残らないまま0件検索が続く。
func TestCheckLocaltimeAgreesWithGo_DetectsLibcFallingBackToUTC(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("modernc の libc が TZ / /etc/localtime を読む musl 転写になるのは linux だけ")
	}
	got := runLocaltimeChildWithTZ(t, ":/nonexistent/gkill_localtime")
	if !strings.HasPrefix(got, "agrees=false ") {
		t.Fatalf("ゾーンファイルが無い libc（UTC）と Asia/Tokyo の Go を一致と判定した: %s", got)
	}
	if !strings.Contains(got, "go=12:09:04/3 ") || !strings.Contains(got, "sqlite=03:09:04/3") {
		t.Fatalf("期待した値の組み合わせではない: %s", got)
	}
}

// 修正の仕組みそのもの: TZ=:<TZif ファイル> と POSIX 固定オフセット文字列のどちらでも
// musl の 'localtime' が Go と揃う。Android の fixTimezone はこの2つを順に使う。
func TestCheckLocaltimeAgreesWithGo_TZFileAndPosixStringAlign(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("modernc の libc が TZ / /etc/localtime を読む musl 転写になるのは linux だけ")
	}
	const tzif = "/usr/share/zoneinfo/Asia/Tokyo"
	if _, err := os.Stat(tzif); err != nil {
		t.Skipf("tzdata が無い環境: %v", err)
	}
	for _, tz := range []string{":" + tzif, "JST-9", "<+09>-9"} {
		got := runLocaltimeChildWithTZ(t, tz)
		if !strings.HasPrefix(got, "agrees=true ") {
			t.Fatalf("TZ=%q で一致しない: %s", tz, got)
		}
	}
}

// 修正の前提そのもの: TZ=:<相対名> は、その名前のファイルが CWD に実在しても musl は zoneinfo ディレクトリ
// （/usr/share/zoneinfo/ 等）でしか探さず、無ければエラーなしで UTC に落ちる。
// fixTimezone が渡すパスが展開済みの絶対パスでなければならない理由（2026-09-16、Termux で既定の
// --gkill_home_dir="$HOME/gkill" を未展開のまま渡して TZ=:$HOME/gkill/tz/localtime になっていた）。
func TestCheckLocaltimeAgreesWithGo_RelativeTZPathFallsBackToUTC(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("modernc の libc が TZ / /etc/localtime を読む musl 転写になるのは linux だけ")
	}
	tzif, err := os.ReadFile("/usr/share/zoneinfo/Asia/Tokyo")
	if err != nil {
		t.Skipf("tzdata が無い環境: %v", err)
	}
	// Termux で実際にできていた形: CWD 直下に文字どおり "$HOME" という名のディレクトリ
	dir := t.TempDir()
	rel := filepath.Join("$HOME", "gkill", "tz", "localtime")
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, tzif, 0o600); err != nil {
		t.Fatal(err)
	}

	got := runLocaltimeChildWithTZInDir(t, ":"+rel, dir)
	if !strings.HasPrefix(got, "agrees=false ") || !strings.Contains(got, "sqlite=03:09:04/3") {
		t.Fatalf("相対名の TZ=:%s で musl が UTC に落ちる前提が崩れた（テストの前提を見直すこと）: %s", rel, got)
	}
	// 同じファイルを絶対パスで渡せば一致する。中身ではなくパスの形の問題であること
	got = runLocaltimeChildWithTZInDir(t, ":"+abs, dir)
	if !strings.HasPrefix(got, "agrees=true ") {
		t.Fatalf("絶対パスの TZ=:%s で一致しない: %s", abs, got)
	}
}
