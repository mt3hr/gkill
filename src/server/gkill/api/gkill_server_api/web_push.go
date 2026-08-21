package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// sendWebPushToTarget は1つの通知先へWebPush送信する。戻り値 shouldDelete=true のとき、
// その通知先は失効（410 Gone）なので呼び出し側が DeleteGkillNotificationTarget する。
//
// 以前は err 非nil でも return せず resp.Body / resp.Status を参照していたため、
// webpush.SendNotification が失敗時に返す nil resp で panic（recoverで500）していた。
// WebPush の失効・不達は日常的に起きるので、この経路は必ず err で早期 return する。
func (g *GkillServerAPI) sendWebPushToTarget(ctx context.Context, subscriptionJSON string, contentJSONb []byte, serverConfig *server_config.ServerConfig) bool {
	s := &webpush.Subscription{}
	if err := json.Unmarshal([]byte(subscriptionJSON), s); err != nil {
		err = fmt.Errorf("error at unmarshal webpush subscription: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		return false
	}
	resp, err := webpush.SendNotification(contentJSONb, s, &webpush.Options{
		Subscriber:      "example@example.com",
		VAPIDPublicKey:  serverConfig.GkillNotificationPublicKey,
		VAPIDPrivateKey: serverConfig.GkillNotificationPrivateKey,
		TTL:             0,
	})
	if err != nil {
		// err 非nil のとき resp は nil。resp を触らずに戻る。
		err = fmt.Errorf("error at send gkill notification: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		return false
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Log(ctx, gkill_log.Debug, "error at defer close", "error", cerr)
		}
	}()
	// 登録解除されていたら呼び出し側で消す
	return resp.StatusCode == http.StatusGone
}

func (g *GkillServerAPI) WebPushUpdatedData(ctx context.Context, userID string, device string, kyouID string) {
	// 通知する
	// 現在のServerConfigを取得する
	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator")
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return
	}

	// 送信対象を取得する
	notificationTargets, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.GetGkillNotificationTargets(ctx, userID, currentServerConfig.GkillNotificationPublicKey)
	if err != nil {
		err = fmt.Errorf("get notification target. in gkill notificator.: %w", err)
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return
	}

	content := &struct {
		IsUpdatedDataNotify bool   `json:"is_updated_data_notify"`
		ID                  string `json:"id"`
	}{
		IsUpdatedDataNotify: true,
		ID:                  kyouID,
	}
	contentJSONb, err := json.Marshal(content)
	if err != nil {
		err = fmt.Errorf("error at marshal webpush content: %w", err)
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return
	}

	for _, notificationTarget := range notificationTargets {
		if g.sendWebPushToTarget(ctx, string(notificationTarget.Subscription), contentJSONb, currentServerConfig) {
			_, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.DeleteGkillNotificationTarget(ctx, notificationTarget.ID)
			if err != nil {
				err = fmt.Errorf("error at delete gkill notification target after got 410 Gone: %w", err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			}
		}
	}
}
