package req_res

type GetIDFKyouByRelativePathRequest struct {
	SessionID string `json:"session_id"`

	TargetID string `json:"target_id"`

	RelativePath string `json:"relative_path"`

	LocaleName string `json:"locale_name"`
}
