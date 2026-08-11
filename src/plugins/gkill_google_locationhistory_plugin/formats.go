package main

import (
	"bytes"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// sniffBytes は形式判定のために読む先頭バイト数。
const sniffBytes = 64 * 1024

// maxInMemoryEntryBytes は丸ごとメモリに載せるパーサが受け入れる上限。
//
// 実データの Timeline Edits.json は展開後 46MB。
// 桁違いに大きいものはZIP爆弾か別物なので、読まずに落とす。
const maxInMemoryEntryBytes = 1 << 30

// gpsFormat は1つの入力フォーマットの認識と読み取り。
//
// 新しいGoogleの書き出し形式は、この表に1行足すだけで対応できるようにしてある。
type gpsFormat struct {
	// ID は file_cache.format・設定画面・テストで使う識別子。
	ID string
	// Label は設定画面に出す表示名。
	Label string
	// Sniff は先頭 sniffBytes バイトを見て、この形式かを判定する。
	// 判定の主体を中身にしているので、ファイル名が変わっていても拾える。
	Sniff func(head []byte) bool
	// Parse はエントリを読んで点を返す。
	// nil なら「認識するが未対応」。設定画面にその旨を出して読み飛ばす。
	Parse func(entry sdk.SourceEntry) ([]rawPoint, error)
}

// gpsFormats は認識するフォーマットの一覧。**順序に意味がある**。
//
// timeline_edits_json を先に置いてあるのは、その中に "locations" のような
// 別形式のキーが現れても records_json に取られないようにするため。
var gpsFormats = []gpsFormat{
	{
		ID:    formatTimelineEdits,
		Label: "タイムライン編集 (Timeline Edits.json)",
		Sniff: func(head []byte) bool { return bytes.Contains(head, []byte(`"timelineEdits"`)) },
		Parse: parseTimelineEdits,
	},
	{
		ID:    formatAndroidTimeline,
		Label: "端末からの書き出し (location-history.json)",
		Sniff: func(head []byte) bool { return bytes.Contains(head, []byte(`"timelinePath"`)) },
		Parse: parseAndroidTimeline,
	},
	{
		ID:    formatRecordsJSON,
		Label: "旧ロケーション履歴 (Records.json)",
		Sniff: func(head []byte) bool {
			if !bytes.Contains(head, []byte(`"locations"`)) {
				return false
			}
			return bytes.Contains(head, []byte(`"latitudeE7"`)) || bytes.Contains(head, []byte(`"timestampMs"`))
		},
		Parse: parseRecordsJSON,
	},
	{
		ID:    formatFitbitGPSCSV,
		Label: "ワークアウトのトラック (gps_location_*.csv)",
		Sniff: func(head []byte) bool {
			line, _, _ := bytes.Cut(head, []byte("\n"))
			lower := strings.ToLower(strings.TrimSpace(string(line)))
			return strings.HasPrefix(lower, "timestamp,latitude,longitude")
		},
		Parse: parseFitbitGPSCSV,
	},
	{
		ID:    formatSemanticLocationHistory,
		Label: "セマンティックロケーション履歴 (YYYY_MONTH.json)",
		Sniff: func(head []byte) bool { return bytes.Contains(head, []byte(`"timelineObjects"`)) },
		// 未対応。
		//
		// 座標の密度があるのは activitySegment の waypointPath / simplifiedRawPath だが、
		// 書き出し時期によって座標の入れ方が違い、手元に検証できる実データが無い。
		// placeVisit.location だけを読むと1日2点程度しか出ず、
		// 「読めているのに中身が薄い」状態になって気づけない。
		// 認識だけして「未対応」と出すほうが正直なので、あえて Parse を nil にしてある。
		Parse: nil,
	},
}

// フォーマットID。
const (
	formatTimelineEdits           = "timeline_edits_json"
	formatAndroidTimeline         = "android_timeline_json"
	formatRecordsJSON             = "records_json"
	formatFitbitGPSCSV            = "fitbit_gps_csv"
	formatSemanticLocationHistory = "semantic_location_history"
)

// formatByID はIDから定義を引く。
var formatByID = buildFormatByID()

func buildFormatByID() map[string]gpsFormat {
	byID := map[string]gpsFormat{}
	for _, format := range gpsFormats {
		byID[format.ID] = format
	}
	return byID
}

// detectFormat はエントリの先頭を見て形式を判定する。
// どれにも当たらなければ空文字を返す。
//
// 伸長は先頭 sniffBytes バイトで打ち切られるので、
// 展開後1GBのエントリでも64KBぶんしか解凍しない。
func detectFormat(entry sdk.SourceEntry) string {
	// 明らかに違うものは開かずに落とす
	lowerName := strings.ToLower(entry.Name)
	if strings.HasSuffix(lowerName, ".csv") && !strings.HasPrefix(lowerName, "gps_location") {
		return ""
	}

	head, err := entry.ReadHead(sniffBytes)
	if err != nil || len(head) == 0 {
		return ""
	}

	for _, format := range gpsFormats {
		if format.Sniff(head) {
			return format.ID
		}
	}
	return ""
}

// readEntryAll はエントリを丸ごと読む。ストリームで読めないパーサ用。
func readEntryAll(entry sdk.SourceEntry) ([]byte, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return sdk.ReadAllLimited(reader, maxInMemoryEntryBytes)
}

// isSupportedFormat は読み取り実装があるかを返す。
func isSupportedFormat(formatID string) bool {
	format, exist := formatByID[formatID]
	return exist && format.Parse != nil
}

// parseByFormat は形式に応じてエントリを読む。
func parseByFormat(formatID string, entry sdk.SourceEntry) ([]rawPoint, error) {
	format, exist := formatByID[formatID]
	if !exist || format.Parse == nil {
		return nil, nil
	}
	return format.Parse(entry)
}
