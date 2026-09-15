package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type ResetPasswordResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	PasswordResetPathWithoutHost string `json:"password_reset_path_without_host"`
}
