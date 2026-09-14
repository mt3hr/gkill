package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type SubmitKFTLTextResponse struct {
	Messages []*message.GkillMessage `json:"messages"`
	Errors   []*message.GkillError   `json:"errors"`

	// Created は確定したレコード。KFTLは1つのテキストから複数のKyouを作るのに、
	// 2026-08-24 まで応答は「記録しました」の1文だけで、件数も種別もIDも返らなかった。
	// 確定は1つの SQLite トランザクション（commit_tx と同じ）なので、失敗したときは何も
	// 残らず空。冪等キーで再送を畳んだときも実行していないので空。
	Created []*SubmitKFTLTextCreated `json:"created"`
}

// SubmitKFTLTextCreated はKFTLが書いた1件。
type SubmitKFTLTextCreated struct {
	ID       string `json:"id"`
	DataType string `json:"data_type"`
	// Updated は新規作成ではなく既存レコードの更新であることを表す（打刻の終了）。
	Updated bool `json:"updated"`
}
