// ˅
package req_res

// ˄

type UpdateReKyouResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UpdatedReKyou *ReKyou `json:"updated_rekyou"`

	UpdatedReKyouKyou *Kyou `json:"updated_rekyou_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
