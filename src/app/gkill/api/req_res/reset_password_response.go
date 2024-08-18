// ˅
package req_res

// ˄

type ResetPasswordResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	PasswordResetPathWithoutHost string `json:"password_reset_path_without_host"`

	// ˅

	// ˄
}

// ˅

// ˄
