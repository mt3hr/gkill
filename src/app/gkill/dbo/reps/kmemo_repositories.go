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

func (k *KmemoRepositories) FindKyous(ctx context.Context, queryJSON string) []*Kyou {
	// ˅

	// ˄
}

func (k *KmemoRepositories) GetKyou(ctx context.Context, id string) *Kyou {
	// ˅

	// ˄
}

func (k *KmemoRepositories) GetKyouHistories(ctx context.Context, id string) []*Kyou {
	// ˅

	// ˄
}

func (k *KmemoRepositories) GetPath(ctx context.Context, id string) string {
	// ˅

	// ˄
}

func (k *KmemoRepositories) UpdateCache(ctx context.Context) {
	// ˅

	// ˄
}

func (k *KmemoRepositories) GetRepName(ctx context.Context) string {
	// ˅

	// ˄
}

func (k *KmemoRepositories) Close(ctx context.Context) {
	// ˅

	// ˄
}

func (k *KmemoRepositories) FindKmemo(ctx context.Context, queryJSON string) []*Kmemo {
	// ˅

	// ˄
}

func (k *KmemoRepositories) GetKmemo(ctx context.Context, id string) *Kmemo {
	// ˅

	// ˄
}

func (k *KmemoRepositories) GetKmemoHistories(ctx context.Context, id string) []*Kmemo {
	// ˅

	// ˄
}

func (k *KmemoRepositories) AddKmemoInfo(ctx context.Context, kmemo *Kmemo) {
	// ˅

	// ˄
}

// ˅

// ˄
