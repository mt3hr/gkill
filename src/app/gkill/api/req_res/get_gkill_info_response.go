package req_res

import (
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/message"
)

type GetGkillInfoResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	UserIsAdmin bool `json:"user_is_admin"`

	CacheClearCountLimit int64 `json:"cache_clear_count_limit"`

	GlobalIP string `json:"global_ip"`

	PrivateIP string `json:"private_ip"`

	Version string `json:"version"`

	BuildTime time.Time `json:"build_time"`

	CommitHash string `json:"commit_hash"`
}
