package req_res

type SubmitKFTLTextRequest struct {
	SessionID  string `json:"session_id"`
	KFTLText   string `json:"kftl_text"`
	LocaleName string `json:"locale_name"`
	// IdempotencyKey は同じ送信の再配送を1回の登録に畳むための任意キー。
	// 空なら冪等判定をしない（従来どおり毎回登録）。Wear のワーカー再送で
	// 結果だけ届かなかったとき、同じキーで再送しても二重登録にならない（監査 S3-wear）。
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}
