// ˅
package req_res

// ˄

type GetApplicationConfigResponse struct {
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
