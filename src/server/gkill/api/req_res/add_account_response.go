package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account"
)

type AddAccountResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	AddedAccountInfo *account.Account `json:"added_account_info"`
}
