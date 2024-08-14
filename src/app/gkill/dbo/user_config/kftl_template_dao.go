// ˅
package user_config

import "context"

// ˄

type KFTLTemplateDAO interface {
	GetAllKFTLTemplates(ctx context.Context) []*KFTLTemplate

	GetKFTLTemplates(ctx context.Context, userID string, device string) []*KFTLTemplate

	AddKFTLTemplate(ctx context.Context, kftlTemplate *KFTLTemplate) bool

	AddKFTLTemplates(ctx context.Context, kftlTemplates []*KFTLTemplate) bool

	UpdateKFTLTemplate(ctx context.Context, kftlTemplate *KFTLTemplate) bool

	DeleteKFTLTemplate(ctx context.Context, id string) bool

	DeleteUsersKFTLTemplates(ctx context.Context, userID string) bool

	// ˅

	// ˄
}

// ˅

// ˄
