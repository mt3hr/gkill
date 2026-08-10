package reps

import (
	"context"
	"slices"
	"sync"
)

// findWarningsCtxKeyType は検索警告コレクタをcontextへ載せるためのキー型
type findWarningsCtxKeyType struct{}

var findWarningsCtxKey = findWarningsCtxKeyType{}

// findWarnings は検索中に起きた「致命的ではない欠落」の記録。
// エラーとして返すと検索全体が失敗扱いになり、クライアントは結果を破棄してしまうため、
// 警告として持ち回り、呼び出し元がメッセージ(GkillMessage)として返す。
type findWarnings struct {
	mu          sync.Mutex
	pluginNames []string
}

// WithFindWarnings は検索警告コレクタを載せたcontextを返します。
// 検索の入口(usecase等)で呼び、検索後に PluginFindWarnings で回収してください。
func WithFindWarnings(ctx context.Context) context.Context {
	return context.WithValue(ctx, findWarningsCtxKey, &findWarnings{})
}

// AppendPluginFindWarning はプラグイン検索の失敗を警告として記録します。
// コレクタが無いcontextでは何もしません(rep直叩き・テスト経路でも安全)。
func AppendPluginFindWarning(ctx context.Context, pluginName string) {
	warnings, ok := ctx.Value(findWarningsCtxKey).(*findWarnings)
	if !ok {
		return
	}
	warnings.mu.Lock()
	defer warnings.mu.Unlock()
	warnings.pluginNames = append(warnings.pluginNames, pluginName)
}

// PluginFindWarnings は記録済みのプラグイン検索失敗(プラグイン名)を返します。
// コレクタが無ければnilを返します。
func PluginFindWarnings(ctx context.Context) []string {
	warnings, ok := ctx.Value(findWarningsCtxKey).(*findWarnings)
	if !ok {
		return nil
	}
	warnings.mu.Lock()
	defer warnings.mu.Unlock()
	return slices.Clone(warnings.pluginNames)
}
