package req_res

type GetIDFFilePathRequest struct {
	SessionID string `json:"session_id"`

	RepName string `json:"rep_name"`

	FileName string `json:"file_name"`

	LocaleName string `json:"locale_name"`
}
