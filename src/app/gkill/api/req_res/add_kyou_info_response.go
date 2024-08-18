// ˅
package req_res

// ˄

type AddKyouInfoResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedKyou *Kyou `json:"added_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
