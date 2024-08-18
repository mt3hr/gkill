// ˅
package req_res

// ˄

type AddKmemoResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedKmemo *Kmemo `json:"added_kmemo"`

	AddedKmemoKyou *Kyou `json:"added_kmemo_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
