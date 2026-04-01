package usecase

import (
	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/dao"
)

// UsecaseContext はユースケース層の依存関係をまとめる構造体
type UsecaseContext struct {
	DAOManager *dao.GkillDAOManager
	FindFilter *api.FindFilter
}

// NewUsecaseContext は新しいUsecaseContextを作成する
func NewUsecaseContext(daoManager *dao.GkillDAOManager, findFilter *api.FindFilter) *UsecaseContext {
	return &UsecaseContext{
		DAOManager: daoManager,
		FindFilter: findFilter,
	}
}
