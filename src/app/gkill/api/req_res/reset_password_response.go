package req_res

import "github.com/mt3hr/gkill/src/app/gkill/api/message"

type ResetPasswordResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	PasswordResetPathWithoutHost string `json:"password_reset_path_without_host"`
}
