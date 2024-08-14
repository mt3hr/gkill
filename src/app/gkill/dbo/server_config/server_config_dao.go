// ˅
package server_config

import "context"

// ˄

type ServerConfigDAO interface {
	GetAllServerConfigs(ctx context.Context) []*ServerConfig

	GetServerConfig(ctx context.Context, device string) *ServerConfig

	AddServerConfig(ctx context.Context, serverConfig *ServerConfig) bool

	UpdateServerConfig(ctx context.Context, serverConfig *ServerConfig) bool

	DeleteServerConfig(ctx context.Context, id string) bool

	// ˅

	// ˄
}

// ˅

// ˄
