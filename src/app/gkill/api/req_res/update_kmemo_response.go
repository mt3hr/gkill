// ˅
package req_res

// ˄

type UpdateKmemoResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UpdatedKmemo *Kmemo `json:"updated_kmemo"`

	UpdatedKyou *Kyou `json:"updated_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
