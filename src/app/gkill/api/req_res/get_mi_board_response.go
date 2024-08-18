// ˅
package req_res

// ˄

type GetMiBoardResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	Boards []*string `json:"boards"`

	// ˅

	// ˄
}

// ˅

// ˄
