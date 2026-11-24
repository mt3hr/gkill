package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

// ExportKyouHTMLResponse は出力した HTML（または ZIP）を返す応答。
//
// **ファイルそのものを body で返さない。** Content-Disposition は src/server に1件も無く、
// クライアントの共通層（gkill_fetch）が JSON 以外の応答を bad_response の合成応答に
// 変えてしまうため、受け取った側がエラーとして扱う。スキルの ZIP 配布が同じ理由で
// base64 を選んでいるので、それに倣う（ADR-0712）。
type ExportKyouHTMLResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	// OutputFormat は要求と同じ ExportOutputFormat*。
	OutputFormat string `json:"output_format"`

	// FileName は保存するときのファイル名（拡張子つき）。
	FileName string `json:"file_name"`

	// HTML は ExportOutputFormatSingle のときだけ入る。
	HTML string `json:"html"`

	// ZipBase64 は ExportOutputFormatSidecar のときだけ入る。
	// HTML と同名フォルダをまとめた ZIP を base64 にしたもの。
	ZipBase64 string `json:"zip_base64"`

	// Partial は予算に収まらず一部を落としたとき真。落としたものは Warnings に入る。
	Partial bool `json:"partial"`

	// Warnings は落としたものや、生成が間に合わなかったサムネイルの知らせ。
	Warnings []string `json:"warnings"`
}
