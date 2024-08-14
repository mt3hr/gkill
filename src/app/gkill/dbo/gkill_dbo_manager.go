// ˅
package dbo

import "github.com/mt3hr/gkill/src/app/gkill/dbo/reps"

// ˄

type GkillDBOManager struct {
	// ˅

	// ˄

	gkillRepositoriesList []*reps.GkillRepositories

	ConfigDAOs *ConfigDAOs

	// ˅

	// ˄
}

func (g *GkillDBOManager) GetRepositories(userID string, device string) (*reps.GkillRepositories, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (g *GkillDBOManager) CloseUserRepositories(userID string, device string) (bool, error) {
	// ˅
	panic("notImplements")
	// ˄
}

// ˅

// ˄
