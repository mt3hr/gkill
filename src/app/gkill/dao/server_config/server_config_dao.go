package server_config

import "context"

type ServerConfigDAO interface {
	GetAllServerConfigs(ctx context.Context) ([]*ServerConfig, error)

	GetServerConfig(ctx context.Context, device string) (*ServerConfig, error)

	AddServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error)

	UpdateServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error)

	UpdateServerConfigs(ctx context.Context, serverConfigs []*ServerConfig) (bool, error)

	DeleteServerConfig(ctx context.Context, device string) (bool, error)

	Close(ctx context.Context) error
}
