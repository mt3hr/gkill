package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/find"

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

// 出力の形。
const (
	// ExportOutputFormatSingle は HTML ファイル1枚。メディアは data: URI で中に入る。
	ExportOutputFormatSingle = "single"
	// ExportOutputFormatSidecar は HTML と同名フォルダの組。ZIP にして返す。
	ExportOutputFormatSidecar = "sidecar"
)

// ExportRykvHTMLRequest は rykv の画面を HTML へ出力する要求。
//
// POST /api/export_rykv_html（wrapAuthRepos）
type ExportRykvHTMLRequest struct {
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
	Columns []*find.FindQuery `json:"columns"`

	// 付随データを含めるか。偽の項目は焼き込まれない。
	IncludeAttachedTags          bool `json:"include_attached_tags"`
	IncludeAttachedTexts         bool `json:"include_attached_texts"`
	IncludeAttachedNotifications bool `json:"include_attached_notifications"`
	IncludeAttachedTimeIss       bool `json:"include_attached_timeiss"`

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
