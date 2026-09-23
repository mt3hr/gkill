package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

// ParseKFTLTextResponse はメモ帳（KFTL）のテキストの解析結果。
//
// 書き間違いは errors ではなく invalid_lines に載る（解析そのものは成功しているので HTTP 200）。
// errors に載るのはリクエスト JSON の不正や設定の取得失敗だけ。
type ParseKFTLTextResponse struct {
	Messages message.GkillMessages `json:"messages"`
	Errors   message.GkillErrors   `json:"errors"`

	// InvalidLines は行別の入力エラー。1件も無ければ空配列（null にしない。
	// クライアントは「空 = 送信してよい」で判定するので、null と [] を区別させない）。
	InvalidLines []*ParseKFTLTextInvalidLine `json:"invalid_lines"`
	// Tags は送信すると付くタグ名（重複なし・出現順）。未知タグの確認に使う。
	Tags []string `json:"tags"`
	// TagGroups は記録ごとのタグの組（組の中は重複なし・出現順、記録の登録順。タグの無い記録は入れない）。
	// Web のメモ帳が保存に成功したあと、組ごとにタグ履歴へ積む。1件も無ければ空配列。
	TagGroups [][]string `json:"tag_groups"`
	// MiBoardNames は Mi / MiReKyou に書かれた板名（空欄は含めない・重複なし・出現順）。
	// 既定板への解決はしない —— 確認ダイアログは利用者が書いたとおりの名前で聞く。
	MiBoardNames []string `json:"mi_board_names"`
	// RecordCount は繰り返しを展開したあとの、書き込みの候補になるリクエスト数。
	RecordCount int `json:"record_count"`
}

// ParseKFTLTextInvalidLine は「おかしな行」1つ。
type ParseKFTLTextInvalidLine struct {
	// LineNumber は1始まりの行番号。0 は「行が分からない」（繰り返しの展開で行が特定できないとき等）。
	LineNumber int `json:"line_number"`
	// LineText は利用者自身が書いたその行のテキスト。
	LineText string `json:"line_text"`
	// Message はローカライズ済みの理由（submit_kftl_text の errors[].error_message と同じ文面）。
	Message string `json:"message"`
}
