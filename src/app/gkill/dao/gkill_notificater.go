package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
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
	cancelFunc      context.CancelFunc
	notification    *reps.Notification
	timer           *time.Timer
	done            bool
}

// 作って通知にそなえて構えます。
// Cancelメソッドからキャセルしてください
func newNotificator(gkillDAOManager *GkillDAOManager, gkillReps *reps.GkillRepositories, notification *reps.Notification) *notificator {
	ctx, cancelFunc := context.WithCancel(context.Background())
	newNotificator := &notificator{
		gkillDAOManager: gkillDAOManager,
		gkillReps:       gkillReps,
		ctx:             ctx,
		cancelFunc:      cancelFunc,
		notification:    notification,
		done:            false,
	}
	go newNotificator.waitAndNotify()
	return newNotificator
}

func (n *notificator) waitAndNotify() {
	ctx := context.Background()
	if n.timer != nil {
		n.timer.Stop()
	}
	// 時間が来たときの通知ハンドラ。
	// まだ通知対象に残っていれば通知する。
	// その後、通知を更新済みに更新し、通知対象から削除する
	if time.Now().Before(n.notification.NotificationTime) {
		// まだだったら時刻まで待機する
		diff := n.notification.NotificationTime.Sub(time.Now())
		<-time.NewTimer(diff).C
	} else {
		return
	}

	// キャンセルされていたら何もせずに返す
	select {
	case <-n.ctx.Done():
		return
	default:
		break
	}

	// Notificationデータを更新する
	updatedNotification := *n.notification
	updatedNotification.IsNotificated = true
	err := n.gkillReps.WriteNotificationRep.AddNotificationInfo(ctx, &updatedNotification)
	if err != nil {
		gkill_log.Debug.Print(err)
		return
	}

	// 通知対象を取得して送信する

	// 現在のServerConfigを取得する
	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := n.gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		gkill_log.Debug.Print(err)
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator.")
		gkill_log.Debug.Print(err)
		return
	}

	// 送信対象を取得する
	userID, err := n.gkillReps.GetUserID(ctx)
	if err != nil {
		err = fmt.Errorf("get user id from gkill reps. in gkill notificator.")
		gkill_log.Debug.Print(err)
		return
	}
	notificationTargets, err := n.gkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.GetGkillNotificationTargets(ctx, userID, currentServerConfig.GkillNotificationPublicKey)
	if err != nil {
		err = fmt.Errorf("get notification target. in gkill notificator.: %w", err)
		gkill_log.Debug.Print(err)
		return
	}

	for _, notificationTarget := range notificationTargets {
		content := &struct {
			Content string `json:"content"`
			URL     string `json:"url"`
		}{
			Content: n.notification.Content,
			URL:     "/kyou?kyou_id=" + n.notification.TargetID,
		}
		contentJSONb, err := json.Marshal(content)
		if err != nil {
			err = fmt.Errorf("error at marshal webpush content: %w", err)
			gkill_log.Debug.Print(err)
			return
		}

		subscription := string(notificationTarget.Subscription)
		s := &webpush.Subscription{}
		json.Unmarshal([]byte(subscription), s)
		resp, err := webpush.SendNotification(contentJSONb, s, &webpush.Options{
			Subscriber:      "example@example.com",
			VAPIDPublicKey:  currentServerConfig.GkillNotificationPublicKey,
			VAPIDPrivateKey: currentServerConfig.GkillNotificationPrivateKey,
			TTL:             0,
		})
		if err != nil {
			err = fmt.Errorf("error at send gkill notification: %w", err)
		}
		defer resp.Body.Close()
	}
	n.done = true
}

func (n *notificator) GetNotificationID() string {
	return n.notification.ID
}

func (n *notificator) Cancel() {
	n.cancelFunc()
	if n.timer != nil {
		n.timer.Stop()
	}
}

type GkillNotificator struct {
	gkillDAOManager *GkillDAOManager
	gkillReps       *reps.GkillRepositories
	notificators    map[string]*notificator
	closed          bool
	m               sync.Mutex
	ticker          *time.Ticker
	ctx             context.Context
	cancelFunc      context.CancelFunc
}

func NewGkillNotificator(gkillDAOManager *GkillDAOManager, gkillReps *reps.GkillRepositories) (*GkillNotificator, error) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	gkillNotificator := &GkillNotificator{
		gkillDAOManager: gkillDAOManager,
		gkillReps:       gkillReps,
		notificators:    map[string]*notificator{},
		closed:          false,
		ticker:          time.NewTicker(time.Hour * 1), // 1時間に1回自動で更新する
		ctx:             ctx,
		cancelFunc:      cancelFunc,
	}
	go gkillNotificator.updateLoopWhenTick()
	return gkillNotificator, nil
}

func (g *GkillNotificator) updateLoopWhenTick() {
loop:
	for {
		err := g.UpdateNotificationTargets(g.ctx)
		if err != nil {
			gkill_log.Debug.Print(err)
		}

		select {
		case <-g.ctx.Done():
			break loop
		case <-g.ticker.C:
		}
	}
}

func (g *GkillNotificator) UpdateNotificationTargets(ctx context.Context) error {
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
	for _, notificator := range g.notificators {
		notificator.Cancel()
	}
	g.notificators = map[string]*notificator{}
	for _, notification := range notifications {
		if notification.IsDeleted || notification.IsNotificated {
			continue
		}
		notificator := newNotificator(g.gkillDAOManager, g.gkillReps, notification)
		if err != nil {
			err = fmt.Errorf("error at new notificator: %w", err)
			return err
		}
		g.notificators[notification.ID] = notificator
	}
	return nil
}

func (g *GkillNotificator) Close(ctx context.Context) error {
	g.cancelFunc()
	g.ticker.Stop()
	g.closed = true
	return nil
}
