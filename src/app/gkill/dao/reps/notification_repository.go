package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
)

type NotificationRepository interface {
	FindNotifications(ctx context.Context, query *find.FindQuery) ([]*Notification, error)

	Close(ctx context.Context) error

	GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error)

	GetNotificationsByTargetID(ctx context.Context, target_id string) ([]*Notification, error)

	GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]*Notification, error)

	UpdateCache(ctx context.Context) error

	GetPath(ctx context.Context, id string) (string, error)

	GetRepName(ctx context.Context) (string, error)

	GetNotificationHistories(ctx context.Context, id string) ([]*Notification, error)

	AddNotificationInfo(ctx context.Context, notification *Notification) error

	UnWrapTyped() ([]NotificationRepository, error)

	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error)
}
