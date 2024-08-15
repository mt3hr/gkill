// ˅
package reps

import "context"

// ˄

type nlogRepositorySQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (n *nlogRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) Close(ctx context.Context) error {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) FindNlog(ctx context.Context, queryJSON string) ([]*Nlog, error) {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) GetNlog(ctx context.Context, id string) (*Nlog, error) {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) GetNlogHistories(ctx context.Context, id string) ([]*Nlog, error) {
	panic("notImplements")
}

func (n *nlogRepositorySQLite3Impl) AddNlogInfo(ctx context.Context, nlog *Nlog) error {
	panic("notImplements")
}

// ˄
