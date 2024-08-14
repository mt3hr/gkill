// ˅
package server_config

import "context"

// ˄

type ServerConfigDAO interface {
	GetAllServerConfigs(ctx context.Context) ([]*ServerConfig, error)

	GetServerConfig(ctx context.Context, device string) (*ServerConfig, error)

	AddServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error)

	UpdateServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error)

	DeleteServerConfig(ctx context.Context, id string) (bool, error)

	// ˅

	// ˄
}

// ˅

// ˄
