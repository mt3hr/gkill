package main

import (
	"net/url"
	"time"
)

const (
	// repName はタイムライン上の rep 名。manifest.json の rep_name と一致させる。
	repName = "VRChat"
	// dataType は manifest.json の data_type。1本で TimeIs / URLog / Kmemo を返すので、
	// 代表の種類（TimeIs）の値にしてある。各 Kyou の data_type は kindDataType で決める（ADR-0317）。
	dataType = "timeis_start"
	// appName はプラグインの識別名。manifest.json の name / executable と一致させる。
	appName = "gkill_plugin_vrchat"
)

// Kyou ごとの data_type。MCP は kmemo と urlog を完全一致で見るので、接頭辞ではなく正確な値にする。
const (
	dataTypeTimeIs = "timeis_start"
	dataTypeURLog  = "urlog"
	dataTypeKmemo  = "kmemo"
)

// 記録に付けるタグ。
const (
	tagWorld  = "VRChat_world"
	tagPlayer = "VRChat_同部屋"
	tagEvent  = "VRChat_event"
)

// 記録の種類。Kyou ID の材料にも使うので、値を変えると全 Kyou の ID が変わる。
const (
	kindWorldTimeIs = "world_timeis"
	kindWorldURLog  = "world_urlog"
	kindPlayer      = "player"
	kindEvent       = "event"
)

// kindDataType は記録の種類から Kyou の data_type を決める。
func kindDataType(kind string) string {
	switch kind {
	case kindWorldURLog:
		return dataTypeURLog
	case kindEvent:
		return dataTypeKmemo
	default:
		return dataTypeTimeIs
	}
}

// worldLaunchURLBase は VRChat 公式サイトの起動用ページ。worldId（と instanceId）を渡すと
// そのワールド（インスタンス）のページが開く。
const worldLaunchURLBase = "https://vrchat.com/home/launch"

// worldURLOf はワールド ID とインスタンスから URLog の URL を作る。
// インスタンスは `~` や `(` を含むので、クエリとしてエスケープする。
func worldURLOf(worldID, instance string) string {
	query := url.Values{}
	query.Set("worldId", worldID)
	if instance != "" {
		query.Set("instanceId", instance)
	}
	return worldLaunchURLBase + "?" + query.Encode()
}

// record は1件の記録（1 Kyou）。1回の起動（1ログファイル）の中で作り切る。
type record struct {
	ID         string
	SessionKey string
	Kind       string
	Title      string
	// Start / End は TimeIs の区間。Kmemo と URLog では Start だけが意味を持つ（記録の時刻）。
	Start time.Time
	End   time.Time
	// URL / Description は URLog の欄。
	URL         string
	Description string
	// Content は Kmemo の本文。
	Content    string
	Tags       []string
	SearchText string
}

// updateTime は Kyou の UpdateTime。
//
// TimeIs は終了時刻にする。プレイ中に運んだログは最後の行で仮に閉じてあり、
// 次の同期で終了が伸びると記録の中身が変わるので、更新時刻も動かしておく。
func (r record) updateTime() time.Time {
	if kindDataType(r.Kind) == dataTypeTimeIs && !r.End.IsZero() {
		return r.End
	}
	return r.Start
}
