package req_res

import "github.com/mt3hr/gkill/src/server/gkill/dao/reps"

type AddURLogRequest struct {
	SessionID string `json:"session_id"`

	URLog reps.URLog `json:"urlog"`

	TXID *string `json:"tx_id"`

	LocaleName string `json:"locale_name"`

	AddedKyou *reps.Kyou `json:"added_kyou"`

	WantResponseKyou bool `json:"want_response_kyou"`

	// SkipFetchMetadata が true のとき、登録前のページ本文取得
	// （空の Title・Description・ThumbnailImage の補完）を行わない。
	// 既定 false = 従来どおり取得する（ブックマークレット等の既存クライアント互換）。
	// MCP の gkill_add_urlog が fetch_metadata:false をここへ写す（MCPレビュー）。
	SkipFetchMetadata bool `json:"skip_fetch_metadata"`

	// SkipFetchFavicon が true のとき、favicon の取得を行わない。既定 false = 取得する。
	SkipFetchFavicon bool `json:"skip_fetch_favicon"`
}
