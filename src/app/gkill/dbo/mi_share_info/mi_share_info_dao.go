// ˅
package mi_share_info

import "context"

// ˄

type MiShareInfoDAO interface {
	GetAllMiShareInfos(ctx context.Context) []*MiShareInfo

	GetMiShareInfos(ctx context.Context, userID string, device string) []*MiShareInfo

	AddMiShareInfo(ctx context.Context, miShareInfo *MiShareInfo) bool

	UpdateMiShareInfo(ctx context.Context, miShareInfo *MiShareInfo) bool

	DeleteMiShareInfo(ctx context.Context, id string) bool

	// ˅

	// ˄
}

// ˅

// ˄
