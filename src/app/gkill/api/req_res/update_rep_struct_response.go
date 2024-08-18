// ˅
package req_res

// ˄

type UpdateRepStructResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	ApplicationConfig *ApplicationConfig `json:"application_config"`

	// ˅

	// ˄
}

// ˅

// ˄
