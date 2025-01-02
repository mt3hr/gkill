package gkill_notification

import "context"

type GkillNotificateTargetDAO interface {
	GetAllGkillNotificationTargets(ctx context.Context) ([]*GkillNotificateTarget, error)

	GetGkillNotificationTargets(ctx context.Context, userID string, publicKey string) ([]*GkillNotificateTarget, error)

	AddGkillNotificationTarget(ctx context.Context, gkillNotificateTarget *GkillNotificateTarget) (bool, error)

	UpdateGkillNotificationTarget(ctx context.Context, gkillNotificateTarget *GkillNotificateTarget) (bool, error)

	DeleteGkillNotificationTarget(ctx context.Context, id string) (bool, error)

	Close(ctx context.Context) error
}
