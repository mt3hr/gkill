// ˅
package req_res

import "github.com/mt3hr/gkill/src/app/gkill/api"

// ˄

type AddAccountResponse struct {
	// ˅

	// ˄

	Messages []*api.GkillMessage

	Errors []*GkillError

	AddedAccountInfo *Account

	// ˅

	// ˄
}

// ˅

// ˄
