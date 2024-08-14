// ˅
package user_config

import "context"

// ˄

type ApplicationConfigDAO interface {
	GetAllApplicationConfigs(ctx context.Context) []*ApplicationConfig

	GetApplicationConfig(ctx context.Context, userID string, device string) *ApplicationConfig

	AddApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) bool

	UpdateApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) bool

	DeleteApplicationConfig(ctx context.Context, userID string, device string) bool

	// ˅

	// ˄
}

// ˅

// ˄
