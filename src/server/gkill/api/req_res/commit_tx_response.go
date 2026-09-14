package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
)

type CommitTxResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	// Committed は確定した全件。確定は1つの SQLite トランザクションなので、
	// 失敗したときは空（何も書かれていない）で、成功したときだけ全件が載る。
	// 形は /api/submit_kftl_text の created[] と同じ。
	Committed []*CommittedRecord `json:"committed"`
}

// CommittedRecord は commit_tx が確定した1件。
type CommittedRecord struct {
	ID       string `json:"id"`
	DataType string `json:"data_type"`
	// Updated は新規作成ではなく既存レコードの新しい版であることを表す（打刻の終了など）。
	Updated bool `json:"updated"`
}
