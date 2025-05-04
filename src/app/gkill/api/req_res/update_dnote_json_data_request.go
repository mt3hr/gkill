package req_res

import "encoding/json"

type UpdateDnoteJSONDataRequest struct {
	SessionID string `json:"session_id"`

	DnoteJSONData json.RawMessage `json:"dnote_json_data"`
}
