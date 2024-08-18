// ˅
package req_res

// ˄

type GetKmemoResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	KmemoHistories []*Kmemo `json:"kmemo_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
