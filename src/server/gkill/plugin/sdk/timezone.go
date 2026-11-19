package sdk

import (
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Android では Go の time.Local が常に UTC になる（time/zoneinfo_android.go の initLocal は
// 端末のゾーンを読まない。TZ 環境変数も見ない）。gkill 本体は main/common の fixTimezone で
// getprop から直しているが、プラグインは別のプロセスなので直っていなかった。
// タイムゾーンを持たない時刻（VRChat のログ、予算の予定日など）を time.Local で読むプラグインは、
// Termux で端末のゾーンとずれた時刻（日本なら9時間前）を返していた。
//
// Run の最初に直すので、プラグインは time.Local を「呼ぶ時点で」読めばよい。
// パッケージ変数に time.Local を写すと、写した時点の UTC のまま残るので注意
// （time.Local はポインタで、直すときはポインタごと差し替える）。ADR-0319。

// fixAndroidTimezone は Android で time.Local を端末のゾーン（persist.sys.timezone）に直す。
// Android 以外では何もしない。直せなければ UTC のまま続ける（起動は止めない）。
func fixAndroidTimezone() {
	if name, ok := fixTimezoneFor(runtime.GOOS, androidZoneName); ok {
		LogDebug("time.Local を端末のゾーンに直しました: %s", name)
	}
}

// fixTimezoneFor は fixAndroidTimezone の本体。OS とゾーン名の読み手を差し替えられるようにしてある（テスト用）。
// 直したら、そのゾーン名と true を返す。
func fixTimezoneFor(goos string, zoneName func() (string, error)) (string, bool) {
	if goos != "android" {
		return "", false
	}
	name, err := zoneName()
	if err != nil {
		LogWarn("端末のタイムゾーンを読めませんでした。時刻は UTC で扱います: %v", err)
		return "", false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		LogWarn("端末のタイムゾーン %q を読み込めませんでした。時刻は UTC で扱います: %v", name, err)
		return "", false
	}
	time.Local = location
	return name, true
}

// androidZoneName は端末のゾーン名（例 Asia/Tokyo）を getprop で読む。
func androidZoneName() (string, error) {
	out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
