// ˅
package req_res

// ˄

type GetServerConfigResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	ServerConfig *ServerConfig `json:"server_config"`

	// ˅

	// ˄
}

// ˅

// ˄
