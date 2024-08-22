// ˅
package api

import (
	"context"

	"github.com/mt3hr/gkill/src/app/gkill/dao"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

// ˄

type FindFilter struct {
	// ˅

	// ˄

	// ˅

	// ˄
}

func (f *FindFilter) FindKyous(gkillDAOManager dao.GkillDAOManager, queryJSON string) ([]*reps.Kyou, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) selectMatchRepsFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) updateCache(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) parseWordsFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) parseTagFilterModeFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) parseTimeIsTagFilterModeFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) getAllKyousWhenDateInRep(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) filterWordsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) filterTagsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) filterPlaingTimeIsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) filterLocationKyous(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (f *FindFilter) sortResultKyous(ctx context.Context, findCtx *FindKyouContext) ([]*GkillError, error) {
	// ˅
	panic("notImplements")
	// ˄
}

// ˅

// ˄
