// ˅
package reps

import "context"

// ˄

type KmemoRepositories struct {
	// ˅

	// ˄

	kmemoRepositories []KmemoRepository

	// ˅

	// ˄
}

func (k *KmemoRepositories) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) UpdateCache(ctx context.Context) error {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) GetRepName(ctx context.Context) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) Close(ctx context.Context) error {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) FindKmemo(ctx context.Context, queryJSON string) ([]*Kmemo, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) GetKmemo(ctx context.Context, id string) (*Kmemo, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) GetKmemoHistories(ctx context.Context, id string) ([]*Kmemo, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (k *KmemoRepositories) AddKmemoInfo(ctx context.Context, kmemo *Kmemo) error {
	// ˅
	panic("notImplements")
	// ˄
}

// ˅

// ˄
