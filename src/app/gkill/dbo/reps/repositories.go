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

func (r *Repositories) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (r *Repositories) Close(ctx context.Context) error {
	// ˅
	panic("notImplements")
	// ˄
}

func (r *Repositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (r *Repositories) UpdateCache(ctx context.Context) error {
	// ˅
	panic("notImplements")
	// ˄
}

func (r *Repositories) GetPath(ctx context.Context, id string) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (r *Repositories) GetRepName(ctx context.Context) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (r *Repositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	// ˅
	panic("notImplements")
	// ˄
}

// ˅

// ˄
