package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
)

type AddAccountResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	AddedAccountInfo *account.Account `json:"added_account_info"`
}
