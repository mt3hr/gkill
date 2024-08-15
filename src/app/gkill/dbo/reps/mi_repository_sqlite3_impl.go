// ˅
package reps

import "context"

// ˄

type miRepositorySQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (m *miRepositorySQLite3Impl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) Close(ctx context.Context) error {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) FindMi(ctx context.Context, queryJSON string) ([]*Mi, error) {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) GetMi(ctx context.Context, id string) (*Mi, error) {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) GetMiHistories(ctx context.Context, id string) ([]*Mi, error) {
	panic("notImplements")
}

func (m *miRepositorySQLite3Impl) AddMiInfo(ctx context.Context, mi *Mi) error {
	panic("notImplements")
}

// ˄
