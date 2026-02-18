package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

type notificator struct {
	gkillDAOManager *GkillDAOManager
	gkillReps       *reps.GkillRepositories
	ctx             context.Context
	notification    *reps.Notification
}

// 作って通知にそなえて構えます。
// キャンセルはctxからやってください
func newNotificator(ctx context.Context, gkillDAOManager *GkillDAOManager, gkillReps *reps.GkillRepositories, notification *reps.Notification) *notificator {
	newNotificator := &notificator{
		ctx:             ctx,
		gkillDAOManager: gkillDAOManager,
		gkillReps:       gkillReps,
		notification:    notification,
	}
	go newNotificator.waitAndNotify()
	return newNotificator
}

func (n *notificator) waitAndNotify() {
	// 時間が来たときの通知ハンドラ。
	// まだ通知対象に残っていれば通知する。
	// その後、通知を更新済みに更新し、通知対象から削除する
	if time.Now().Before(n.notification.NotificationTime) {
		// まだだったら時刻まで待機する
		diff := time.Until(n.notification.NotificationTime)

		select {
		case <-n.ctx.Done():
			return
		case <-time.After(diff):
		}
	}

	notificationCtx := context.Background()

	// Notificationデータを更新する
	updatedNotification := *n.notification
	updatedNotification.IsNotificated = true
	updatedNotification.UpdateTime = time.Now()
	updatedNotification.UpdateUser = "gkill_notificator"
	err := n.gkillReps.WriteNotificationRep.AddNotificationInfo(notificationCtx, updatedNotification)
	if err != nil {
		slog.Log(n.ctx, gkill_log.Error, "error", "error", err)
		return
	}

	// 通知対象を取得して送信する

	// 現在のServerConfigを取得する
	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := n.gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(notificationCtx)
	if err != nil {
		slog.Log(n.ctx, gkill_log.Error, "error", "error", err)
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator")
		slog.Log(n.ctx, gkill_log.Error, "error", "error", err)
		return
	}

	// 送信対象を取得する
	userID, err := n.gkillReps.GetUserID(notificationCtx)
	if err != nil {
		err = fmt.Errorf("get user id from gkill reps. in gkill notificator")
		slog.Log(n.ctx, gkill_log.Error, "error", "error", err)
		return
	}
	notificationTargets, err := n.gkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.GetGkillNotificationTargets(notificationCtx, userID, currentServerConfig.GkillNotificationPublicKey)
	if err != nil {
		err = fmt.Errorf("get notification target. in gkill notificator.: %w", err)
		slog.Log(n.ctx, gkill_log.Error, "error", "error", err)
		return
	}

	for _, notificationTarget := range notificationTargets {
		content := &struct {
			IsNotification bool      `json:"is_notification"`
			Content        string    `json:"content"`
			URL            string    `json:"url"`
			Time           time.Time `json:"time"`
		}{
			IsNotification: true,
			Content:        n.notification.Content,
			URL:            "/kyou?kyou_id=" + n.notification.TargetID,
			Time:           n.notification.NotificationTime,
		}
		contentJSONb, err := json.Marshal(content)
		if err != nil {
			err = fmt.Errorf("error at marshal webpush content: %w", err)
			slog.Log(n.ctx, gkill_log.Error, "error", "error", err)
			return
		}

		subscription := string(notificationTarget.Subscription)
		s := &webpush.Subscription{}
		_ = json.Unmarshal([]byte(subscription), s)
		resp, err := webpush.SendNotification(contentJSONb, s, &webpush.Options{
			Subscriber:      "example@example.com",
			VAPIDPublicKey:  currentServerConfig.GkillNotificationPublicKey,
			VAPIDPrivateKey: currentServerConfig.GkillNotificationPrivateKey,
			TTL:             0,
		})
		if err != nil {
			err = fmt.Errorf("error at send gkill notification: %w", err)
			slog.Log(n.ctx, gkill_log.Debug, "error", "error", err)
		}
		if resp.Body != nil {
			defer func() {
				err := resp.Body.Close()
				if err != nil {
					slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
				}
			}()
		}
		// 登録解除されていたらDBから消す
		if resp.Status == "410 Gone" {
			_, err := n.gkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.DeleteGkillNotificationTarget(notificationCtx, notificationTarget.ID)
			if err != nil {
				err = fmt.Errorf("error at delete gkill notification target after got 410 Gone: %w", err)
				slog.Log(n.ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	}
}

type GkillNotificator struct {
	gkillDAOManager        *GkillDAOManager
	gkillReps              *reps.GkillRepositories
	notificators           map[string]*notificator
	notificationServiceCtx context.Context
	notificationCtx        context.Context
	cancelFunc             context.CancelFunc
}

func NewGkillNotificator(ctx context.Context, gkillDAOManager *GkillDAOManager, gkillReps *reps.GkillRepositories) (*GkillNotificator, error) {
	gkillNotificator := &GkillNotificator{
		gkillDAOManager:        gkillDAOManager,
		gkillReps:              gkillReps,
		notificators:           map[string]*notificator{},
		notificationServiceCtx: ctx,
	}
	go gkillNotificator.updateLoopWhenTick()
	return gkillNotificator, nil
}

func (g *GkillNotificator) updateLoopWhenTick() {
	for {
		err := g.UpdateNotificationTargets(context.Background())
		if err != nil {
			slog.Log(g.notificationServiceCtx, gkill_log.Error, "error", "error", err)
		}

		select {
		case <-g.notificationServiceCtx.Done():
			g.cancelFunc()
			return
		case <-time.After(1 * time.Hour):
		}
	}
}

func (g *GkillNotificator) UpdateNotificationTargets(ctx context.Context) error {
	// 現在のServerConfigを取得する
	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", err)
		return err
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator")
		slog.Log(ctx, gkill_log.Error, "error", "error", err)
		return err
	}
	if !currentServerConfig.UseGkillNotification {
		return nil
	}

	// 30分前から1時間30分あとを範囲として取得する
	startTime, endTime := time.Now().Add(time.Minute*30*-1), time.Now().Add(time.Minute*90)

	// 最新のNotificationを取得する
	notifications, err := g.gkillReps.NotificationReps.GetNotificationsBetweenNotificationTime(ctx, startTime, endTime)
	if err != nil {
		repName, _ := g.gkillReps.NotificationReps.GetRepName(ctx)
		err = fmt.Errorf("error at get notifications between notification time at %s: %w", repName, err)
		return err
	}

	// 今あるnotificatorを全部キャンセルして新しく作る
	if g.cancelFunc != nil {
		g.cancelFunc()
	}
	g.notificationCtx, g.cancelFunc = context.WithCancel(g.notificationServiceCtx)

	g.notificators = map[string]*notificator{}
	for _, notification := range notifications {
		if notification.IsDeleted || notification.IsNotificated {
			continue
		}
		notificator := newNotificator(g.notificationCtx, g.gkillDAOManager, g.gkillReps, &notification)
		g.notificators[notification.ID] = notificator
	}
	return nil
}

func (g *GkillNotificator) Close(ctx context.Context) error {
	if g.cancelFunc != nil {
		g.cancelFunc()
	}
	return nil
}
