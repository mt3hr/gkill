// ˅
package req_res

// ˄

type AddLantanaResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedLantana *Lantana `json:"added_lantana"`

	AddedLantanaKyou *Kyou `json:"added_lantana_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
