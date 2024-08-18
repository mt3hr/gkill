// ˅
package req_res

// ˄

type UpdateMiResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UpdatedMi *Mi `json:"updated_mi"`

	UpdatedMiKyou *Kyou `json:"updated_mi_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
