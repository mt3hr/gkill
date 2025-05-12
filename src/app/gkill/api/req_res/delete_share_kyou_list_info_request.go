package req_res

type DeleteShareKyouListInfoRequest struct {
	SessionID string `json:"session_id"`

	ShareKyouListInfo *ShareKyouListInfo `json:"share_kyou_list_info"`
}
