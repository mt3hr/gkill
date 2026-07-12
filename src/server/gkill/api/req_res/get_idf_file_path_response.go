package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetIDFFilePathResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	// ファイルの絶対パス。同一マシンからのリクエストでないとき、
	// または対象ファイルが見つからないときは空文字。
	FilePath string `json:"file_path"`

	// FilePathが解決できたかどうか
	Exists bool `json:"exists"`
}
