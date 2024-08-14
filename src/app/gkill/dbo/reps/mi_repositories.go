// ˅
package reps

import "context"

// ˄

type MiRepositories struct {
	// ˅

	// ˄

	miRepositories []MiRepository

	// ˅

	// ˄
}

func (m *MiRepositories) FindKyous(ctx context.Context, queryJSON string) []*Kyou {
	// ˅

	// ˄
}

func (m *MiRepositories) GetKyou(ctx context.Context, id string) *Kyou {
	// ˅

	// ˄
}

func (m *MiRepositories) GetKyouHistories(ctx context.Context, id string) []*Kyou {
	// ˅

	// ˄
}

func (m *MiRepositories) GetPath(ctx context.Context, id string) string {
	// ˅

	// ˄
}

func (m *MiRepositories) UpdateCache(ctx context.Context) {
	// ˅

	// ˄
}

func (m *MiRepositories) GetRepName(ctx context.Context) string {
	// ˅

	// ˄
}

func (m *MiRepositories) Close(ctx context.Context) {
	// ˅

	// ˄
}

func (m *MiRepositories) FindMi(ctx context.Context, queryJSON string) []*Mi {
	// ˅

	// ˄
}

func (m *MiRepositories) GetMi(ctx context.Context, id string) *Mi {
	// ˅

	// ˄
}

func (m *MiRepositories) GetMiHistories(ctx context.Context, id string) []*Mi {
	// ˅

	// ˄
}

func (m *MiRepositories) AddMiInfo(ctx context.Context, mi *Mi) {
	// ˅

	// ˄
}

// ˅

// ˄
