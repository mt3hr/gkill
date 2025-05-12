package share_kyou_info

import "context"

type ShareKyouInfoDAO interface {
	GetAllKyouShareInfos(ctx context.Context) ([]*ShareKyouInfo, error)

	GetKyouShareInfos(ctx context.Context, userID string, device string) ([]*ShareKyouInfo, error)

	GetKyouShareInfo(ctx context.Context, sharedID string) (*ShareKyouInfo, error)

	AddKyouShareInfo(ctx context.Context, shareKyouInfo *ShareKyouInfo) (bool, error)

	UpdateKyouShareInfo(ctx context.Context, shareKyouInfo *ShareKyouInfo) (bool, error)

	DeleteKyouShareInfo(ctx context.Context, shareID string) (bool, error)

	Close(ctx context.Context) error
}
