package mi_share_info

import "context"

type MiShareInfoDAO interface {
	GetAllMiShareInfos(ctx context.Context) ([]*MiShareInfo, error)

	GetMiShareInfos(ctx context.Context, userID string, device string) ([]*MiShareInfo, error)

	GetMiShareInfo(ctx context.Context, sharedID string) (*MiShareInfo, error)

	AddMiShareInfo(ctx context.Context, miShareInfo *MiShareInfo) (bool, error)

	UpdateMiShareInfo(ctx context.Context, miShareInfo *MiShareInfo) (bool, error)

	DeleteMiShareInfo(ctx context.Context, id string) (bool, error)

	Close(ctx context.Context) error
}
