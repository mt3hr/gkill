// Package req_res は全エンドポイントのリクエスト/レスポンス構造体(クライアント側TypeScriptクラスとのJSONミラー)。
package req_res

type Account struct {
	UserID string `json:"user_id"`

	IsAdmin bool `json:"is_admin"`

	IsEnable bool `json:"is_enable"`
}
