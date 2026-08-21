package gkill_server_api

import (
	"context"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
)

// M-1: WebPush 送信失敗（webpush.SendNotification が nil resp を返す）で panic しないこと。
// 以前は err 非nil でも resp.Body / resp.Status を参照して nil deref panic していた。
func TestSendWebPushToTarget_DoesNotPanicOnSendFailure(t *testing.T) {
	g := &GkillServerAPI{}
	serverConfig := &server_config.ServerConfig{
		GkillNotificationPublicKey:  "",
		GkillNotificationPrivateKey: "",
	}
	ctx := context.Background()

	t.Run("空のsubscriptionでも panic しない", func(t *testing.T) {
		// エンドポイントが無いので SendNotification は nil resp + err を返す。
		got := g.sendWebPushToTarget(ctx, `{}`, []byte(`{"x":1}`), serverConfig)
		if got {
			t.Error("failed send should return shouldDelete=false")
		}
	})

	t.Run("壊れたsubscription JSONでも panic しない", func(t *testing.T) {
		got := g.sendWebPushToTarget(ctx, `{not json`, []byte(`{"x":1}`), serverConfig)
		if got {
			t.Error("invalid subscription should return shouldDelete=false")
		}
	})
}
