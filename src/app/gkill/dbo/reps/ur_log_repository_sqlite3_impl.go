// ˅
package reps

import "context"

// ˄

type urlogRepositorySQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (u *urlogRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) Close(ctx context.Context) error {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) FindURLog(ctx context.Context, queryJSON string) ([]*URLog, error) {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) GetURLog(ctx context.Context, id string) (*URLog, error) {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) GetURLogHistories(ctx context.Context, id string) ([]*URLog, error) {
	panic("notImplements")
}

func (u *urlogRepositorySQLite3Impl) AddURLogInfo(ctx context.Context, urlog *URLog) error {
	panic("notImplements")
}

// ˄
