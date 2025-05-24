package req_res

type CommitTxRequest struct {
	SessionID string `json:"session_id"`

	TXID string `json:"tx_id"`
}
