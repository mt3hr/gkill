package req_res

type DiscardTxRequest struct {
	SessionID string `json:"session_id"`

	TXID string `json:"tx_id"`
}
