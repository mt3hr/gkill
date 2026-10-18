package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
)

// DeleteKyouCascadeResponse は連鎖削除の結果です。
type DeleteKyouCascadeResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	// Deleted は消したもの全部（起点・付随データ・参照元）です。失敗したときは空で、何も消えていません。
	Deleted []*DeletedRecord `json:"deleted"`
}

// DeletedRecord は連鎖削除で消した1件です。DataType はエンティティ名（kmemo / mi / mirekyou / tag / text /
// notification / rekyou / idf …）で、射影名ではありません。
type DeletedRecord struct {
	ID string `json:"id"`

	DataType string `json:"data_type"`
}
