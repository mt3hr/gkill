package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
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

// notifiedVersion は配信を済ませた通知の新しい版を作る。作成の欄は元の版のまま、更新の欄は
// 「アプリ gkill_notificator・利用者と端末はこのサーバの値・時刻 now」を刻む。
// 以前は更新者の欄に gkill_notificator（アプリ名）を入れ、更新アプリ・更新端末は前の版のまま残していた。
func notifiedVersion(notification reps.Notification, now time.Time, userID string, device string) reps.Notification {
	notification.IsNotificated = true
	notification.UpdateTime = now
	notification.UpdateApp = "gkill_notificator"
	notification.UpdateDevice = device
	notification.UpdateUser = userID
	return notification
}

// getCurrentServerConfig は有効なサーバ設定（EnableThisDevice）を返す。無ければエラー。
func (n *notificator) getCurrentServerConfig(ctx context.Context) (*server_config.ServerConfig, error) {
	serverConfigs, err := n.gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error at get all server configs: %w", err)
	}
	var currentServerConfig *server_config.ServerConfig
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		return nil, fmt.Errorf("current server config is not found. in gkill notificator")
	}
	return currentServerConfig, nil
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

	// 配信済みの版に刻む利用者と端末。送信先の引き当てにも使うので、書き込みより前に取る。
	// 取れなくても配信済みにはする（未配信のまま残すと配信をやり直し続ける）。送信は従来どおり止める。
	currentServerConfig, serverConfigErr := n.getCurrentServerConfig(notificationCtx)
	userID, userIDErr := n.gkillReps.GetUserID(notificationCtx)
	device := ""
	if currentServerConfig != nil {
		device = currentServerConfig.Device
	}

	// Notificationデータを更新する
	updatedNotification := notifiedVersion(*n.notification, time.Now(), userID, device)
	err := n.gkillReps.WriteNotificationRep.AddNotificationInfo(notificationCtx, updatedNotification)
	if err != nil {
		slog.Log(n.ctx, gkill_log.Error, "error at add notification info", "error", fmt.Sprintf("%q", err))
		return
	}

	// キャッシュへ入れる RepName は**書き込み先rep**に合わせる。
	// updatedNotification は GetNotificationsBetweenNotificationTime で読んだ
	// 取得元repの名前を持ったままなので、そのまま入れるとキャッシュ表の REP_NAME が
	// 実体（WriteNotificationRep）と食い違う。取れなければ空にする（空は安全側）
	if writeRepName, repNameErr := n.gkillReps.WriteNotificationRep.GetRepName(notificationCtx); repNameErr == nil {
		updatedNotification.RepName = writeRepName
	} else {
		updatedNotification.RepName = ""
	}

	// 読み取りはキャッシュrepしか見ないので、ここを飛ばすと次のUpdateCache(最大1分)まで
	// IsNotificated=false のまま読まれる。その間にPWAが古い応答をキャッシュし直すと、
	// 更新IDの再通知が来ないぶん「未通知」が恒久的に焼き付く。
	// 書き込み経路の作法は usecase/notification.go の AddNotification と同じ
	err = n.gkillReps.WriteThroughNotificationCache(notificationCtx, updatedNotification)
	if err != nil {
		err = fmt.Errorf("error at write through notification cache id = %s: %w", updatedNotification.ID, err)
		slog.Log(n.ctx, gkill_log.Error, "error at write through notification cache id", "error", fmt.Sprintf("%q", err))
	}

	repName, err := n.gkillReps.WriteNotificationRep.GetRepName(notificationCtx)
	if err != nil {
		err = fmt.Errorf("error at get rep name id = %s: %w", updatedNotification.ID, err)
		slog.Log(n.ctx, gkill_log.Error, "error at get rep name id", "error", fmt.Sprintf("%q", err))
	} else {
		latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              updatedNotification.IsDeleted,
			TargetID:                               updatedNotification.ID,
			TargetIDInData:                         &updatedNotification.TargetID,
			DataUpdateTime:                         updatedNotification.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		n.gkillReps.SetLatestDataRepositoryAddress(updatedNotification.ID, latestDataRepositoryAddress)

		_, err = n.gkillReps.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(notificationCtx, latestDataRepositoryAddress)
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for notification id = %s: %w", updatedNotification.ID, err)
			slog.Log(n.ctx, gkill_log.Error, "error at add or update latest data repository address for notification id", "error", fmt.Sprintf("%q", err))
		}
	}

	// 通知対象を取得して送信する

	// 現在のServerConfig（書き込みの前に取ったもの）
	if serverConfigErr != nil {
		slog.Log(n.ctx, gkill_log.Error, "error at get current server config in gkill notificator", "error", fmt.Sprintf("%q", serverConfigErr))
		return
	}

	// 送信対象を取得する
	if userIDErr != nil {
		err = fmt.Errorf("error at get user id from gkill reps in gkill notificator: %w", userIDErr)
		slog.Log(n.ctx, gkill_log.Error, "error at get user id from gkill reps in gkill notificator", "error", fmt.Sprintf("%q", err))
		return
	}
	notificationTargets, err := n.gkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.GetGkillNotificationTargets(notificationCtx, userID, currentServerConfig.GkillNotificationPublicKey)
	if err != nil {
		err = fmt.Errorf("get notification target. in gkill notificator.: %w", err)
		slog.Log(n.ctx, gkill_log.Error, "get notification target. in gkill notificator", "error", fmt.Sprintf("%q", err))
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
			slog.Log(n.ctx, gkill_log.Error, "error at marshal webpush content", "error", fmt.Sprintf("%q", err))
			return
		}

		subscription := string(notificationTarget.Subscription)
		s := &webpush.Subscription{}
		if err := json.Unmarshal([]byte(subscription), s); err != nil {
			slog.Log(n.ctx, gkill_log.Warn, "error at unmarshal webpush subscription", "error", fmt.Sprintf("%q", err))
			continue
		}
		resp, err := webpush.SendNotification(contentJSONb, s, &webpush.Options{
			Subscriber:      "example@example.com",
			VAPIDPublicKey:  currentServerConfig.GkillNotificationPublicKey,
			VAPIDPrivateKey: currentServerConfig.GkillNotificationPrivateKey,
			TTL:             0,
		})
		if err != nil {
			err = fmt.Errorf("error at send gkill notification: %w", err)
			slog.Log(n.ctx, gkill_log.Warn, "error at send gkill notification", "error", fmt.Sprintf("%q", err))
			continue
		}
		if resp.Body != nil {
			err := resp.Body.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at close response body", "error", fmt.Sprintf("%q", err))
			}
		}
		// 登録解除されていたらDBから消す
		if resp.Status == "410 Gone" {
			_, err := n.gkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.DeleteGkillNotificationTarget(notificationCtx, notificationTarget.ID)
			if err != nil {
				err = fmt.Errorf("error at delete gkill notification target after got 410 Gone: %w", err)
				slog.Log(n.ctx, gkill_log.Warn, "error at delete gkill notification target after got 410 Gone", "error", fmt.Sprintf("%q", err))
			}
		}
	}
}

type GkillNotificator struct {
	gkillDAOManager        *GkillDAOManager
	gkillReps              *reps.GkillRepositories
	notificationServiceCtx context.Context
	cancelService          context.CancelFunc
	updateLoopDone         chan struct{}

	// targetsMutex は通知対象の作り直し（1時間ごとの定期更新と、通知を追加・更新したハンドラからの呼び出し）と
	// Close を直列にする。並走すると cancelFunc を上書きし合い、取り消されない通知が残って二重に送る。
	targetsMutex    sync.Mutex
	notificators    map[string]*notificator
	notificationCtx context.Context
	cancelFunc      context.CancelFunc
}

func NewGkillNotificator(ctx context.Context, gkillDAOManager *GkillDAOManager, gkillReps *reps.GkillRepositories) (*GkillNotificator, error) {
	serviceCtx, cancelService := context.WithCancel(ctx)
	gkillNotificator := &GkillNotificator{
		gkillDAOManager:        gkillDAOManager,
		gkillReps:              gkillReps,
		notificators:           map[string]*notificator{},
		notificationServiceCtx: serviceCtx,
		cancelService:          cancelService,
		updateLoopDone:         make(chan struct{}),
	}
	go gkillNotificator.updateLoopWhenTick()
	return gkillNotificator, nil
}

func (g *GkillNotificator) updateLoopWhenTick() {
	defer close(g.updateLoopDone)
	for {
		err := g.UpdateNotificationTargets(g.notificationServiceCtx)
		if err != nil && g.notificationServiceCtx.Err() == nil {
			slog.Log(g.notificationServiceCtx, gkill_log.Error, "error at update notification targets", "error", fmt.Sprintf("%q", err))
		}

		select {
		case <-g.notificationServiceCtx.Done():
			return
		case <-time.After(1 * time.Hour):
		}
	}
}

func (g *GkillNotificator) UpdateNotificationTargets(ctx context.Context) error {
	g.targetsMutex.Lock()
	defer g.targetsMutex.Unlock()
	// 閉じたあとは設定 DAO を触らない（GkillDAOManager.Close が通知を閉じたあとで nil にする）
	if err := g.notificationServiceCtx.Err(); err != nil {
		return fmt.Errorf("gkill notificator is closed: %w", err)
	}

	// 現在のServerConfigを取得する
	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		slog.Log(ctx, gkill_log.Error, "error at get all server configs", "error", fmt.Sprintf("%q", err))
		return err
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator")
		slog.Log(ctx, gkill_log.Error, "current server config is not found. in gkill notificator", "error", fmt.Sprintf("%q", err))
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

// Close は定期更新を止め、その goroutine が抜けるまで待ってから返る。
// 待たずに返ると、GkillDAOManager.Close が設定 DAO を nil にしたあとで定期更新が
// それを参照して nil ポインタ参照で落ちる（goroutine 内なので recover できずプロセスごと落ちる）。
// 配信待ちの通知は ctx の取り消しで止まるが、配信を始めたものは待たない。
func (g *GkillNotificator) Close(ctx context.Context) error {
	g.cancelService()
	select {
	case <-g.updateLoopDone:
	case <-ctx.Done():
		return fmt.Errorf("error at wait for notification update loop: %w", ctx.Err())
	}

	g.targetsMutex.Lock()
	defer g.targetsMutex.Unlock()
	if g.cancelFunc != nil {
		g.cancelFunc()
	}
	return nil
}
