package req_res

// ParseKFTLTextRequest はメモ帳（KFTL）のテキストを解析だけする要求。何も書かない。
//
// Web のメモ帳が打鍵のたびに投げて「おかしな行」のピンク表示に使い、保存の直前にも投げて
// 未知タグ・未知板名の確認に使う（ADR-0507）。解析は SubmitKFTLTextRequest と同じ
// prepareRequests を通るので、ここで通った入力が送信で弾かれることは無い。
type ParseKFTLTextRequest struct {
	SessionID  string `json:"session_id"`
	KFTLText   string `json:"kftl_text"`
	LocaleName string `json:"locale_name"`
}
