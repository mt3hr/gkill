package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetIDFKyouByRelativePathResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	// 見つからなかった場合は空文字
	KyouID string `json:"kyou_id"`
}
