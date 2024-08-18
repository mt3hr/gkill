// ˅
package req_res

// ˄

type UpdateLantanaResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UpdatedLantana *Lantana `json:"updated_lantana"`

	UpdatedLantanaKyou *Kyou `json:"updated_lantana_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
