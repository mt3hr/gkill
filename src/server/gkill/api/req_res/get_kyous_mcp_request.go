package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/find"

type GetKyousMCPRequest struct {
	SessionID  string          `json:"session_id"`
	Query      *find.FindQuery `json:"query"`
	LocaleName string          `json:"locale_name"`
	Limit      int             `json:"limit"`      // default 50
	Cursor     string          `json:"cursor"`     // ISO-8601, 空=先頭から
	MaxSizeMB  float64         `json:"max_size_mb"` // default 1.0
}
