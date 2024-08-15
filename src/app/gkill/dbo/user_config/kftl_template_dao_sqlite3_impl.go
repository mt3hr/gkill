// ˅
package user_config

import "context"

// ˄

type kftlTemplateDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (k *kftlTemplateDAOSQLite3Impl) GetAllKFTLTemplates(ctx context.Context) ([]*KFTLTemplate, error) {
	panic("notImplements")
}

func (k *kftlTemplateDAOSQLite3Impl) GetKFTLTemplates(ctx context.Context, userID string, device string) ([]*KFTLTemplate, error) {
	panic("notImplements")
}

func (k *kftlTemplateDAOSQLite3Impl) AddKFTLTemplate(ctx context.Context, kftlTemplate *KFTLTemplate) (bool, error) {
	panic("notImplements")
}

func (k *kftlTemplateDAOSQLite3Impl) AddKFTLTemplates(ctx context.Context, kftlTemplates []*KFTLTemplate) (bool, error) {
	panic("notImplements")
}

func (k *kftlTemplateDAOSQLite3Impl) UpdateKFTLTemplate(ctx context.Context, kftlTemplate *KFTLTemplate) (bool, error) {
	panic("notImplements")
}

func (k *kftlTemplateDAOSQLite3Impl) DeleteKFTLTemplate(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

func (k *kftlTemplateDAOSQLite3Impl) DeleteUsersKFTLTemplates(ctx context.Context, userID string) (bool, error) {
	panic("notImplements")
}

// ˄
