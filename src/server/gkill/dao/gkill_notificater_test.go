package dao

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
)

// 配信を済ませた通知の新しい版は、作成の欄を元の版のまま持ち、更新の欄に
// 「アプリ gkill_notificator・このサーバの利用者と端末・配信した時刻」を刻む。
// 以前は更新者の欄に gkill_notificator（アプリ名）を入れ、更新アプリ・更新端末は前の版のまま残していた。
func TestNotifiedVersionStampsUpdateFields(t *testing.T) {
	created := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	notifiedAt := created.Add(2 * time.Hour)
	original := reps.Notification{
		ID: "notification-1", TargetID: "target-1", Content: "通知の本文", IsNotificated: false,
		NotificationTime: created.Add(time.Hour),
		CreateTime:       created, CreateApp: "create_app", CreateDevice: "create_device", CreateUser: "create_user",
		UpdateTime: created, UpdateApp: "update_app", UpdateDevice: "update_device", UpdateUser: "update_user",
	}

	got := notifiedVersion(original, notifiedAt, "testuser", "testdevice")

	if !got.IsNotificated {
		t.Error("配信済みの印が付いていない")
	}
	if got.CreateApp != "create_app" || got.CreateDevice != "create_device" || got.CreateUser != "create_user" || !got.CreateTime.Equal(created) {
		t.Errorf("作成の欄が変わった: app=%q device=%q user=%q time=%v", got.CreateApp, got.CreateDevice, got.CreateUser, got.CreateTime)
	}
	if got.UpdateApp != "gkill_notificator" || got.UpdateDevice != "testdevice" || got.UpdateUser != "testuser" || !got.UpdateTime.Equal(notifiedAt) {
		t.Errorf("更新の欄 = app=%q device=%q user=%q time=%v, want gkill_notificator / testdevice / testuser / %v",
			got.UpdateApp, got.UpdateDevice, got.UpdateUser, got.UpdateTime, notifiedAt)
	}
	if got.ID != original.ID || got.TargetID != original.TargetID || got.Content != original.Content || !got.NotificationTime.Equal(original.NotificationTime) {
		t.Errorf("通知の中身が変わった: %+v", got)
	}
	if original.IsNotificated || original.UpdateApp != "update_app" {
		t.Error("元の版を書き換えている（値で受けて新しい版を返すこと）")
	}
}

// blockingServerConfigDAO は GetAllServerConfigs を ctx が切れるまで返さないスタブ。
// 埋め込みが nil なので、テストで呼ばない他のメソッドを実装する必要がない。
type blockingServerConfigDAO struct {
	server_config.ServerConfigDAO
	entered chan struct{}
	m       sync.Mutex
	calls   int
	inCall  bool
}

func (b *blockingServerConfigDAO) GetAllServerConfigs(ctx context.Context) ([]*server_config.ServerConfig, error) {
	b.m.Lock()
	b.calls++
	b.inCall = true
	b.m.Unlock()
	select {
	case b.entered <- struct{}{}:
	default:
	}
	<-ctx.Done()
	b.m.Lock()
	b.inCall = false
	b.m.Unlock()
	return nil, ctx.Err()
}

func (b *blockingServerConfigDAO) state() (int, bool) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.calls, b.inCall
}

// Close は定期更新の goroutine が DAO から抜けて止まるまで待つ。
// GkillDAOManager.Close は通知を閉じたあとで設定 DAO を nil にするので、Close が待たずに返ると
// 定期更新が閉じた DAO を参照して nil ポインタ参照でテストのプロセスごと落ちていた
// （GetRepositories が通知を作るので、rep を組み立てて Close するテストのどれでも起きうる）。
func TestGkillNotificatorCloseWaitsForUpdateLoop(t *testing.T) {
	stub := &blockingServerConfigDAO{entered: make(chan struct{}, 1)}
	manager := &GkillDAOManager{ConfigDAOs: &ConfigDAOs{ServerConfigDAO: stub}}

	notificator, err := NewGkillNotificator(context.Background(), manager, nil)
	if err != nil {
		t.Fatalf("NewGkillNotificator failed: %v", err)
	}
	select {
	case <-stub.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("定期更新がサーバ設定を読みに来ない")
	}

	if err := notificator.Close(context.Background()); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if _, inCall := stub.state(); inCall {
		t.Fatal("Close が返ったあとも定期更新がサーバ設定の DAO を使っている")
	}

	// 閉じたあとにハンドラから呼ばれても DAO を触らない
	if err := notificator.UpdateNotificationTargets(context.Background()); err == nil {
		t.Error("閉じたあとの UpdateNotificationTargets がエラーを返さない")
	}
	if calls, _ := stub.state(); calls != 1 {
		t.Errorf("GetAllServerConfigs の呼び出し = %d 回, want 1（閉じたあとに読みに行っている）", calls)
	}
}
