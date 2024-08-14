// ˅
package reps

import "context"

// ˄

type Repositories struct {
	// ˅

	// ˄

	repositories []Repository

	// ˅

	// ˄
}

func (r *Repositories) FindKyous(ctx context.Context, queryJSON string) []*Kyou {
	// ˅

	// ˄
}

func (r *Repositories) Close(ctx context.Context) {
	// ˅

	// ˄
}

func (r *Repositories) GetKyou(ctx context.Context, id string) *Kyou {
	// ˅

	// ˄
}

func (r *Repositories) UpdateCache(ctx context.Context) {
	// ˅

	// ˄
}

func (r *Repositories) GetPath(ctx context.Context, id string) string {
	// ˅

	// ˄
}

func (r *Repositories) GetRepName(ctx context.Context) string {
	// ˅

	// ˄
}

func (r *Repositories) GetKyouHistories(ctx context.Context, id string) []*Kyou {
	// ˅

	// ˄
}

// ˅

// ˄
