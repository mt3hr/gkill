// ˅
package dao

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

// ˄

type GkillDAOManager struct {
	// ˅

	// ˄

	gkillRepositoriesList []*reps.GkillRepositories

	ConfigDAOs *ConfigDAOs

	// ˅

	// ˄
}

func (g *GkillDAOManager) GetRepositories(userID string, device string) (*reps.GkillRepositories, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (g *GkillDAOManager) CloseUserRepositories(userID string, device string) (bool, error) {
	// ˅
	panic("notImplements")
	// ˄
}

// ˅

// ˄
