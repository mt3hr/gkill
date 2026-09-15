package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetIDFKyouByRelativePathResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	// 見つからなかった場合は空文字
	KyouID string `json:"kyou_id"`
}
