// ˅
package user_config

import "context"

// ˄

type RepositoryDAO interface {
	GetAllRepositories(ctx context.Context) []*Repository

	GetRepositories(ctx context.Context, userID string, device string) []*Repository

	AddRepository(ctx context.Context, repository *Repository) bool

	UpdateRepository(ctx context.Context, repository *Repository) bool

	DeleteRepository(ctx context.Context, id string) bool

	// ˅

	// ˄
}

// ˅

// ˄
