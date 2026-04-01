package gkill_server_api

import (
	"context"

	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

type contextKey string

const authContextKey contextKey = "gkill_auth_context"

// AuthContext はミドルウェアで認証済みのユーザー情報を保持する
type AuthContext struct {
	Account      *account.Account
	UserID       string
	Device       string
	Repositories *reps.GkillRepositories // Auth-onlyルートではnil
}

// AuthFromContext はコンテキストからAuthContextを取得する。未設定の場合はnilを返す。
func AuthFromContext(ctx context.Context) *AuthContext {
	val, _ := ctx.Value(authContextKey).(*AuthContext)
	return val
}

func contextWithAuth(ctx context.Context, auth *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, auth)
}
