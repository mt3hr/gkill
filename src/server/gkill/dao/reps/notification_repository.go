package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// NotificationRepository はKyouに付随する通知のリポジトリが満たす契約です。
//
// NotificationはKyou本体ではなくKyouに付随するデータなので、Kyouを返す Repository とは
// 別系統のインターフェースになっており、FindKyous ではなく FindNotifications を持ちます。
// どのKyouへの通知かは Notification.TargetID が指し、発火予定時刻は NotificationTime、
// 通知済みかどうかは IsNotificated が持ちます。
// 実装の層構成（sqlite3 / cached / temp / 集約 NotificationRepositories）は Repository と同じです。
type NotificationRepository interface {
	// FindNotifications は検索条件に一致するNotificationを返します。
	// 契約は Repository.FindKyous を参照。戻り値はマップではなくスライスです。
	// 集約(NotificationRepositories)は Repository.FindKyous と同じキー規則で畳み込み、
	// query.IncludeDeletedData が false なら削除済みを除外します。
	FindNotifications(ctx context.Context, query *find.FindQuery) ([]Notification, error)

	// Close は契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// GetNotification は id に対応するNotificationを1件返します。
	// 契約は Repository.GetKyou を参照（見つからない場合は (nil, nil)）。
	GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error)

	// GetNotificationsByTargetID は target_id のKyouに付いているNotificationを返します。
	//
	// Tag/Textと違い最新版に絞らず全バージョンを返し、削除済み(IsDeleted)も除外しません。
	// 集約はUpdateTimeの降順に並べます。
	// 1件も付いていなければ空スライスを返します（エラーではありません）。
	GetNotificationsByTargetID(ctx context.Context, target_id string) ([]Notification, error)

	// GetNotificationsBetweenNotificationTime は NotificationTime が
	// startTime 以上 endTime 以下（両端を含む）のNotificationを返します。
	// 比較はローカルタイムに揃えて行います。
	//
	// UpdateTime ではなく NotificationTime での絞り込みなので、最新版に絞らず
	// 全バージョンを返し、削除済み・通知済みも除外しません。発火対象の選別
	// （IsDeleted / IsNotificated の判定）は呼び出し側の GkillNotificator が行います。
	GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]Notification, error)

	// UpdateCache は契約は Repository.UpdateCache を参照。
	UpdateCache(ctx context.Context) error
	// LastUpdateCacheChanged は直近の UpdateCache で下層の実DBに変化があったかを返します。
	// 契約は Repository.UpdateCache を参照。
	LastUpdateCacheChanged() bool

	// GetLatestDataRepositoryAddress は契約は Repository.GetLatestDataRepositoryAddress を参照。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetPath は契約は Repository.GetPath を参照。
	// Notificationはファイル実体を持たないため、id の有無にかかわらず
	// NotificationのDBファイルのパスを返します（id が非空のときだけ絶対パスに直します）。
	GetPath(ctx context.Context, id string) (string, error)

	// GetRepName は契約は Repository.GetRepName を参照。
	GetRepName(ctx context.Context) (string, error)

	// GetNotificationHistories は id の全バージョンを返します。
	// 契約は Repository.GetKyouHistories を参照。集約はUpdateTimeの降順に並べます。
	GetNotificationHistories(ctx context.Context, id string) ([]Notification, error)

	// AddNotificationInfo はNotificationを1件追加します。追記専用なので、更新は同一IDの
	// 新UPDATE_TIME版、削除は IsDeleted=true 版の追加になります。
	// 通知済みへの変更も IsNotificated=true の新バージョン追加で表します。
	//
	// Tag/Textと違い内容の空チェックはありません。
	// 呼び出し先は書き込み用の単一リポジトリ（WriteNotificationRep など）で、
	// 集約(NotificationRepositories)は未実装エラーを返します。
	// キャッシュ実装はキャッシュDBにだけ書くので、実DBへの書き込みとは別に呼ばれます。
	AddNotificationInfo(ctx context.Context, notification Notification) error

	// UnWrapTyped は契約は Repository.UnWrap を参照。
	// 戻り値は NotificationRepository のスライスです。
	UnWrapTyped() ([]NotificationRepository, error)
}
