// ˅
package reps

import "context"

// ˄

type kmemoRepositorySQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (k *kmemoRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) Close(ctx context.Context) error {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) FindKmemo(ctx context.Context, queryJSON string) ([]*Kmemo, error) {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) GetKmemo(ctx context.Context, id string) (*Kmemo, error) {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) GetKmemoHistories(ctx context.Context, id string) ([]*Kmemo, error) {
	panic("notImplements")
}

func (k *kmemoRepositorySQLite3Impl) AddKmemoInfo(ctx context.Context, kmemo *Kmemo) error {
	panic("notImplements")
}

// ˄
