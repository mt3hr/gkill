// ˅
package req_res

// ˄

type AddReKyouResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedReKyou *ReKyou `json:"added_re_kyou"`

	AddedReKyouKyou *Kyou `json:"added_rekyou_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
