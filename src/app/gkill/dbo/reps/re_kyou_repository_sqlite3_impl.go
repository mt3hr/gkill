// ˅
package reps

import "context"

// ˄

type reKyouRepositorySQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (r *reKyouRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) Close(ctx context.Context) error {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) FindReKyou(ctx context.Context, queryJSON string) ([]*ReKyou, error) {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) GetReKyou(ctx context.Context, id string) (*ReKyou, error) {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) GetReKyouHistories(ctx context.Context, id string) ([]*ReKyou, error) {
	panic("notImplements")
}

func (r *reKyouRepositorySQLite3Impl) AddReKyouInfo(ctx context.Context, rekyou *ReKyou) error {
	panic("notImplements")
}

// ˄
