package dao

import (
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
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
