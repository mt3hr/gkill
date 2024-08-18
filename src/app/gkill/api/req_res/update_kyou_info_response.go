// ˅
package req_res

// ˄

type UpdateKyouInfoResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UpdatedKyou *Kyou `json:"updated_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
