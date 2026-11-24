package req_res

import (
	"encoding/json"
	"fmt"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// ExportKyouHTMLColumn は出力する列1本。
//
// **クライアントが送ってきた JSON を手放さないこと。** find.FindQuery には
// query_id / keywords / *_in_sidebar / is_focus_kyou_in_list_view が無く、
// ここで型に落とすと焼き込みから消える。**列の同一性は query_id** なので、
// 落とすと出力先で列を引けず、2列以上の出力では全列が先頭列に一致する
// （例外もエラーも出ないので、1列で試しているうちは露見しない）。
//
// 収集には型に落とした Query を使い、焼き込みには Raw をそのまま使う。
// こうすると、収集の途中で Query を触っても焼き込みには影響しない
// （焼いた条件がサイドバーの生成する条件と食い違うと、その列が空になる）。
type ExportKyouHTMLColumn struct {
	// Query は収集に使う検索条件。
	Query find.FindQuery
	// Raw はクライアントが送ってきた JSON そのまま。焼き込むのはこちら。
	Raw json.RawMessage
}

// UnmarshalJSON は生の JSON を控えてから型へ落とす。
func (c *ExportKyouHTMLColumn) UnmarshalJSON(data []byte) error {
	c.Raw = append(json.RawMessage(nil), data...)
	if err := json.Unmarshal(data, &c.Query); err != nil {
		return fmt.Errorf("error at parse export rykv html column: %w", err)
	}
	return nil
}

// MarshalJSON は控えた生の JSON をそのまま返す（無ければ型から組む）。
func (c ExportKyouHTMLColumn) MarshalJSON() ([]byte, error) {
	if len(c.Raw) != 0 {
		return c.Raw, nil
	}
	return json.Marshal(&c.Query)
}

// BakedQueryJSON は出力 HTML へ焼き込む列の条件を返す。
func (c ExportKyouHTMLColumn) BakedQueryJSON() (json.RawMessage, error) {
	baked, err := c.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json.RawMessage(baked), nil
}

// IsNull は JSON の null が来たかを返す。
func (c ExportKyouHTMLColumn) IsNull() bool {
	return len(c.Raw) == 0 || string(c.Raw) == "null"
}

// メディア（画像・動画・添付ファイル）の埋め込み方。
const (
	// ExportMediaModeThumbnail はサムネイルだけを埋め込む。
	ExportMediaModeThumbnail = "thumbnail"
	// ExportMediaModeFull は原寸も埋め込む。件数しだいで容易に巨大になる。
	ExportMediaModeFull = "full"
	// ExportMediaModeNone はメディアを埋め込まない。
	// このとき ExportOutputFormatSidecar は選べない（フォルダが空になるため）。
	ExportMediaModeNone = "none"
)

// メディアの予算の既定。利用者がダイアログで変えられる。
const (
	// ExportDefaultMediaBudgetBytes はメディアと添付ファイルの合計の既定。
	ExportDefaultMediaBudgetBytes int64 = 500 * 1024 * 1024
	// ExportDefaultMediaItemLimitBytes は1件あたりの上限の既定。
	ExportDefaultMediaItemLimitBytes int64 = 100 * 1024 * 1024
)

// 出力の形。
const (
	// ExportOutputFormatSingle は HTML ファイル1枚。メディアは data: URI で中に入る。
	ExportOutputFormatSingle = "single"
	// ExportOutputFormatSidecar は HTML と同名フォルダの組。ZIP にして返す。
	ExportOutputFormatSidecar = "sidecar"
)

// ExportKyouHTMLRequest は rykv の画面を HTML へ出力する要求。
//
// POST /api/export_kyou_html（wrapAuthRepos）
type ExportKyouHTMLRequest struct {
	SessionID string `json:"session_id"`

	// LocaleName はエラーメッセージの言語であり、出力 HTML へ焼き込む言語でもある。
	LocaleName string `json:"locale_name"`

	// Title は出力 HTML の表題。
	Title string `json:"title"`

	// Columns は出力する列の検索条件。画面に出ている順で並べる。
	//
	// **サイドバーが生成するのと同じ形で送ること**（チェック済みの全タグと "no tags"、
	// 全 rep）。tags / reps の既定は非 nil の空配列 = 0件指定であって「絞らない」ではないので、
	// 欠けた条件を焼くと、出力先のサイドバーが初めに生成した条件と食い違った瞬間に
	// その列が空になる。
	Columns []ExportKyouHTMLColumn `json:"columns"`

	// 付随データを含めるか。偽の項目は焼き込まれない。
	IncludeAttachedTags          bool `json:"include_attached_tags"`
	IncludeAttachedTexts         bool `json:"include_attached_texts"`
	IncludeAttachedNotifications bool `json:"include_attached_notifications"`
	IncludeAttachedTimeIss       bool `json:"include_attached_timeiss"`

	// MediaBudgetBytes はメディアと添付ファイルに使える生バイトの合計。
	// 0 のときは ExportDefaultMediaBudgetBytes。base64 は素のバイト数の 4/3 になるので、
	// 出来上がる HTML はこれより大きくなる。
	MediaBudgetBytes int64 `json:"media_budget_bytes"`

	// MediaItemLimitBytes はメディア・添付ファイル1件あたりの上限。
	// 0 のときは ExportDefaultMediaItemLimitBytes。1件で合計を食い潰すのを防ぐ。
	MediaItemLimitBytes int64 `json:"media_item_limit_bytes"`

	// IncludeGPSLog は位置情報を含めるか。
	IncludeGPSLog bool `json:"include_gps_log"`

	// IncludePluginContent はプラグインの本文を含めるか。
	// 同一プラグインへの要求は直列化されるので、件数を増やすと待ちが伸びる。
	IncludePluginContent bool `json:"include_plugin_content"`

	// 出力 HTML に出す画面。偽にすると、そのボタンごと出ない。
	IncludeDnote   bool `json:"include_dnote"`
	IncludeMap     bool `json:"include_map"`
	IncludeSidebar bool `json:"include_sidebar"`

	// MediaMode は ExportMediaMode* のいずれか。
	MediaMode string `json:"media_mode"`

	// OutputFormat は ExportOutputFormat* のいずれか。
	OutputFormat string `json:"output_format"`
}
