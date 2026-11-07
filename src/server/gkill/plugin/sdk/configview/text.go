package configview

import (
	"encoding/json"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Prose は設定画面の文言を HTML にする。
//
// 文言は Go の文字列リテラルに書くので、HTML を直接書けると便利だが、そうするとデータ（パス・エラー文）を
// 同じ欄に流し込んだときにエスケープされない。そこで文言の欄はすべてこの関数を通し、
// 描くのは `code`（インラインコード）と **強調** の2つだけにする。それ以外は html.EscapeString と同じ。
//
//   - バッククォートで囲んだ部分は <code>。中の ** は強調にしない（`**` のような記号そのものを書くため）
//   - ** で囲んだ部分は <strong>。行をまたいでもよい
//   - 対が無いバッククォート・** は文字のまま
func Prose(s string) string {
	var sb strings.Builder
	rest := s
	for {
		open := strings.IndexByte(rest, '`')
		if open < 0 {
			break
		}
		closeAt := strings.IndexByte(rest[open+1:], '`')
		if closeAt < 0 {
			break
		}
		sb.WriteString(prosePlain(rest[:open]))
		if closeAt == 0 {
			// 空のコード（``）は文字のまま
			sb.WriteString("``")
		} else {
			sb.WriteString(`<code>` + html.EscapeString(rest[open+1:open+1+closeAt]) + `</code>`)
		}
		rest = rest[open+1+closeAt+1:]
	}
	sb.WriteString(prosePlain(rest))
	return sb.String()
}

var boldPattern = regexp.MustCompile(`\*\*([^*]+?)\*\*`)

// prosePlain はコード以外の部分をエスケープし、**強調** だけを描く。
func prosePlain(s string) string {
	if s == "" {
		return ""
	}
	return boldPattern.ReplaceAllString(html.EscapeString(s), `<strong>$1</strong>`)
}

// Count は件数を3桁区切りで返す。統計表の値と進捗の件数に使う。
func Count(n int) string {
	digits := strconv.Itoa(n)
	negative := strings.HasPrefix(digits, "-")
	digits = strings.TrimPrefix(digits, "-")

	var sb strings.Builder
	for i, r := range digits {
		if i != 0 && (len(digits)-i)%3 == 0 {
			sb.WriteByte(',')
		}
		sb.WriteRune(r)
	}
	if negative {
		return "-" + sb.String()
	}
	return sb.String()
}

// FormatUnix は Unix 秒を表示用の文字列にする。0 以下は "-"（未実行）。
func FormatUnix(unix int64) string {
	if unix <= 0 {
		return "-"
	}
	return FormatTime(time.Unix(unix, 0))
}

// FormatTime は時刻を表示用の文字列にする。ゼロ値は "-"（未実行）。
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

// SampleJSON は config.json の記入例を返す。既定の設定（Handler.DefaultConfig と同じ値）を渡す。
// 自動生成される内容そのものを見せる。整形できなければ "{}"。
func SampleJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// SourceDirsNote は source_dirs の書き方の説明文（Prose 記法）を組む。
// 取り込み元を source_dirs で指定するプラグインが共通で使い、「マッチしたものをどう扱うか」の1文（matched）と
// プラグイン固有の補足（extra。既定値や追加キーの説明）だけを差し込む。
func SourceDirsNote(matched string, extra ...string) string {
	var sb strings.Builder
	sb.WriteString("`source_dirs` は**配列で複数指定**できます(1つなら文字列でも可)。")
	sb.WriteString("ワイルドカード `*` `**` `?` `[]` が使えます — ")
	sb.WriteString(matched)
	sb.WriteString("。先頭の `~` と環境変数(`$HOME` など)も展開されます")
	sb.WriteString("(ただしgkillをWindowsサービスで動かしている場合は実行アカウントのホームになるため、絶対パスが確実です)。")
	for _, s := range extra {
		sb.WriteString(s)
	}
	sb.WriteString("変更は次の検索から反映されます(gkillの再起動は不要)。")
	return sb.String()
}
