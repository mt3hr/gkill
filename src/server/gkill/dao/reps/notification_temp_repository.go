package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// NotificationTempRepository は、まだ確定していないNotificationを置いておく一時リポジトリが
// 満たす契約です。
//
// 行は (txID, userID, device) でスコープされます。トランザクション中の書き込みは
// AddNotificationInfo でここに溜まり、/api/commit_tx が GetNotificationsByTXID で取り出して
// 本体のリポジトリへ書き写し、/api/discard_tx が DeleteByTXID で捨てます。
// 実体は利用者ごとの一時DB（既定はインメモリ）上のNOTIFICATION表ひとつです。
// 未確定の通知は GkillNotificator の発火対象には入りません。
type NotificationTempRepository interface {
	// FindNotifications の契約は NotificationRepository.FindNotifications を参照。
	//
	// ただしsqlite3実装は基底のnotification実装へ構造体変換で委譲しており、変換後は
	// 一時DBのハンドルではなく "notification_temp" というファイル名でDBを開き直すため、
	// 一時表は読めずエラーになります。
	// 未確定データの取り出しには GetNotificationsByTXID を使ってください。
	FindNotifications(ctx context.Context, query *find.FindQuery) ([]Notification, error)

	// Close の契約は Repository.Close を参照。
	// 一時DBは全一時repで共有しているため閉じてはならず、sqlite3実装はDBを閉じません。
	// ただし自分でLockしたうえで同じミューテックスをLockする基底実装へ委譲するので、
	// 呼ぶとデッドロックします。呼び出し元はありません。
	Close(ctx context.Context) error

	// GetNotification の契約は NotificationRepository.GetNotification を参照。
	// FindNotifications と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error)

	// GetNotificationsByTargetID の契約は NotificationRepository.GetNotificationsByTargetID を参照。
	// FindNotifications と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetNotificationsByTargetID(ctx context.Context, target_id string) ([]Notification, error)

	// GetNotificationsBetweenNotificationTime の契約は
	// NotificationRepository.GetNotificationsBetweenNotificationTime を参照。
	// FindNotifications と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]Notification, error)

	// UpdateCache の契約は Repository.UpdateCache を参照。
	// 一時repはキャッシュも変更検知も持たないため、sqlite3実装では何もしません。
	UpdateCache(ctx context.Context) error

	// GetLatestDataRepositoryAddress の契約は Repository.GetLatestDataRepositoryAddress を参照。
	// 未確定データは最新版の所在に含めないため、sqlite3実装は常にエラーを返します。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetPath の契約は Repository.GetPath を参照。
	// 一時repは自分のファイルを持たないため、sqlite3実装は常にエラーを返します。
	GetPath(ctx context.Context, id string) (string, error)

	// GetRepName の契約は Repository.GetRepName を参照。
	// sqlite3実装は固定値 "notification_temp" を返します。
	GetRepName(ctx context.Context) (string, error)

	// GetNotificationHistories の契約は NotificationRepository.GetNotificationHistories を参照。
	// FindNotifications と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetNotificationHistories(ctx context.Context, id string) ([]Notification, error)

	// AddNotificationInfo は notification を (txID, userID, device) 付きで一時表に追記します。
	//
	// 本体のリポジトリと同じ追記専用で、更新は同一IDの新しいUPDATE_TIME版の追加、
	// 削除は IsDeleted=true 版の追加、通知済みへの変更は IsNotificated=true 版の追加として
	// 表現します。内容の空チェックはありません。
	// 一意制約もないので、同じIDを何度追記してもエラーにはなりません。
	AddNotificationInfo(ctx context.Context, notification Notification, txID string, userID string, device string) error

	// GetNotificationsByTXID は txID・userID・device の3つすべてに一致する未確定Notificationを返します。
	//
	// 一致する行が無ければ空スライスを返します（エラーではありません）。
	// RepName は一時rep名で埋めます。最新版への絞り込みも削除済み・通知済みの除外もしないので、
	// そのトランザクションで追記された行がそのまま全部返ります。
	// /api/commit_tx はこれで取り出した結果を本体のリポジトリへ書き写します。
	GetNotificationsByTXID(ctx context.Context, txID string, userID string, device string) ([]Notification, error)

	// DeleteByTXID は (txID, userID, device) に一致する行を一時表から物理削除します。
	// 追記専用の本体リポジトリと違い、ここは実際に行を消します。
	// 一致する行が無くてもエラーにはなりません。
	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	// UnWrapTyped は UnWrap の型付き版で、NotificationTempRepository のまま平坦化します。
	// 一時repはラッパではないので、sqlite3実装は自分自身1件だけを返します。
	UnWrapTyped() ([]NotificationTempRepository, error)
}
