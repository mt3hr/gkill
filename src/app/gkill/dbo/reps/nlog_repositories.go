// ˅
package reps

import "context"

// ˄

type NlogRepositories struct {
	// ˅

	// ˄

	nlogRepositories []NlogRepository

	// ˅

	// ˄
}

func (n *NlogRepositories) FindKyous(ctx context.Context, queryJSON string) []*Kyou {
	// ˅

	// ˄
}

func (n *NlogRepositories) GetKyou(ctx context.Context, id string) *Kyou {
	// ˅

	// ˄
}

func (n *NlogRepositories) GetKyouHistories(ctx context.Context, id string) []*Kyou {
	// ˅

	// ˄
}

func (n *NlogRepositories) GetPath(ctx context.Context, id string) string {
	// ˅

	// ˄
}

func (n *NlogRepositories) UpdateCache(ctx context.Context) {
	// ˅

	// ˄
}

func (n *NlogRepositories) GetRepName(ctx context.Context) string {
	// ˅

	// ˄
}

func (n *NlogRepositories) Close(ctx context.Context) {
	// ˅

	// ˄
}

func (n *NlogRepositories) FindNlog(ctx context.Context, queryJSON string) []*Nlog {
	// ˅

	// ˄
}

func (n *NlogRepositories) GetNlog(ctx context.Context, id string) *Nlog {
	// ˅

	// ˄
}

func (n *NlogRepositories) GetNlogHistories(ctx context.Context, id string) []*Nlog {
	// ˅

	// ˄
}

func (n *NlogRepositories) AddNlogInfo(ctx context.Context, nlog *Nlog) {
	// ˅

	// ˄
}

// ˅

// ˄
