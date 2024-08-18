// ˅
package req_res

import "github.com/mt3hr/gkill/src/app/gkill/api"

// ˄

type AddAccountResponse struct {
	// ˅

	// ˄

	Messages []*api.GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedAccountInfo *Account `json:"added_account_info"`

	// ˅

	// ˄
}

// ˅

// ˄
