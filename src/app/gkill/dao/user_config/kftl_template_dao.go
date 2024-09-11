// ˅
package user_config

import "context"

// ˄

type KFTLTemplateDAO interface {
	GetAllKFTLTemplates(ctx context.Context) ([]*KFTLTemplate, error)

	GetKFTLTemplates(ctx context.Context, userID string, device string) ([]*KFTLTemplate, error)

	AddKFTLTemplate(ctx context.Context, kftlTemplate *KFTLTemplate) (bool, error)

	AddKFTLTemplates(ctx context.Context, kftlTemplates []*KFTLTemplate) (bool, error)

	UpdateKFTLTemplate(ctx context.Context, kftlTemplate *KFTLTemplate) (bool, error)

	DeleteKFTLTemplate(ctx context.Context, id string) (bool, error)

	DeleteUsersKFTLTemplates(ctx context.Context, userID string) (bool, error)

	Close(ctx context.Context) error

	// ˅

	// ˄
}

// ˅

// ˄
