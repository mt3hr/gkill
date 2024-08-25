// ˅
package dao

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gorilla/mux"
	"github.com/mattn/go-zglob"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

// ˄

type GkillDAOManager struct {
	// ˅

	// ˄

	gkillRepositories map[string]map[string]*reps.GkillRepositories

	ConfigDAOs *ConfigDAOs

	// ˅

	router    *mux.Router
	autoIDF   *bool
	IDFIgnore []string

	// ˄
}

func NewGkillDAOManager(autoIDF bool) *GkillDAOManager {
	return &GkillDAOManager{
		router:  &mux.Router{},
		autoIDF: &autoIDF,
		IDFIgnore: []string{
			".kyou",
			".nomedia",
			"desktop.ini",
			"thumbnails",
			".thumbnails",
			"Thumbs.db",
			"steam_autocloud.vdf",
			".DS_Store",
			".localized",
		},
	}
}

func (g *GkillDAOManager) GetRouter() *mux.Router {
	return g.router
}

func (g *GkillDAOManager) GetRepositories(userID string, device string) (*reps.GkillRepositories, error) {
	// ˅
	ctx := context.TODO()

	// nilだったら初期化する
	if g.gkillRepositories == nil {
		g.gkillRepositories = map[string]map[string]*reps.GkillRepositories{}
	}

	// すでに存在していればそれを、存在していなければ作っていれる
	repositoriesInDevice, existInUsers := g.gkillRepositories[userID]
	if !existInUsers {
		g.gkillRepositories[userID] = map[string]*reps.GkillRepositories{}
		repositoriesInDevice, _ = g.gkillRepositories[userID]
	}

	repositories, existInDevice := repositoriesInDevice[device]
	if !existInDevice {
		// なかったら作っていれる
		repositories = reps.NewGkillRepositories(userID)

		repositoriesDefine, err := g.ConfigDAOs.RepositoryDAO.GetRepositories(ctx, userID, device)
		if err != nil {
			err = fmt.Errorf("error at get repositories user=%s device=%s: %w", userID, device, err)
			return nil, err
		}
		for _, rep := range repositoriesDefine {
			if !rep.IsEnable {
				continue
			}
			rep.File = os.ExpandEnv(rep.File)
			matchFiles, _ := zglob.Glob(rep.File)
			sort.Strings(matchFiles)
			for _, filename := range matchFiles {
				parentDir := filepath.Dir(filename)
				err := os.MkdirAll(parentDir, os.ModePerm)
				if err != nil {
					err = fmt.Errorf("error at make directory %s: %w", parentDir)
					return nil, err
				}

				switch rep.Type {
				case "kmemo":
					kmemoRep, err := reps.NewKmemoRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.KmemoReps = append(repositories.KmemoReps, kmemoRep)
					repositories.Reps = append(repositories.Reps, kmemoRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteKmemoRep.GetPath(ctx, "")
						newPath, _ := kmemoRep.GetPath(ctx, "")
						if repositories.WriteKmemoRep != nil {
							err := fmt.Errorf("error conflict write kmemo rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteKmemoRep = kmemoRep
					}
				case "urlog":
					urlogRep, err := reps.NewURLogRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.URLogReps = append(repositories.URLogReps, urlogRep)
					repositories.Reps = append(repositories.Reps, urlogRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteURLogRep.GetPath(ctx, "")
						newPath, _ := urlogRep.GetPath(ctx, "")
						if repositories.WriteURLogRep != nil {
							err := fmt.Errorf("error conflict write urlog rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteURLogRep = urlogRep
					}
				case "timeis":
					timeisRep, err := reps.NewTimeIsRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.TimeIsReps = append(repositories.TimeIsReps, timeisRep)
					repositories.Reps = append(repositories.Reps, timeisRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteTimeIsRep.GetPath(ctx, "")
						newPath, _ := timeisRep.GetPath(ctx, "")
						if repositories.WriteTimeIsRep != nil {
							err := fmt.Errorf("error conflict write timeis rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteTimeIsRep = timeisRep
					}
				case "mi":
					miRep, err := reps.NewMiRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.MiReps = append(repositories.MiReps, miRep)
					repositories.Reps = append(repositories.Reps, miRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteMiRep.GetPath(ctx, "")
						newPath, _ := miRep.GetPath(ctx, "")
						if repositories.WriteMiRep != nil {
							err := fmt.Errorf("error conflict write mi rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteMiRep = miRep
					}
				case "nlog":
					nlogRep, err := reps.NewNlogRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.NlogReps = append(repositories.NlogReps, nlogRep)
					repositories.Reps = append(repositories.Reps, nlogRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteNlogRep.GetPath(ctx, "")
						newPath, _ := nlogRep.GetPath(ctx, "")
						if repositories.WriteNlogRep != nil {
							err := fmt.Errorf("error conflict write nlog rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteNlogRep = nlogRep
					}
				case "lantana":
					lantanaRep, err := reps.NewLantanaRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.LantanaReps = append(repositories.LantanaReps, lantanaRep)
					repositories.Reps = append(repositories.Reps, lantanaRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteLantanaRep.GetPath(ctx, "")
						newPath, _ := lantanaRep.GetPath(ctx, "")
						if repositories.WriteLantanaRep != nil {
							err := fmt.Errorf("error conflict write lantana rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteLantanaRep = lantanaRep
					}
				case "tag":
					tagRep, err := reps.NewTagRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.TagReps = append(repositories.TagReps, tagRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteTagRep.GetPath(ctx, "")
						newPath, _ := tagRep.GetPath(ctx, "")
						if repositories.WriteTagRep != nil {
							err := fmt.Errorf("error conflict write tag rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteTagRep = tagRep
					}
				case "text":
					textRep, err := reps.NewTextRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.TextReps = append(repositories.TextReps, textRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteTextRep.GetPath(ctx, "")
						newPath, _ := textRep.GetPath(ctx, "")
						if repositories.WriteTextRep != nil {
							err := fmt.Errorf("error conflict write text rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteTextRep = textRep
					}
				case "rekyou":
					reKyouRep, err := reps.NewReKyouRepositorySQLite3Impl(ctx, filename, repositories)
					if err != nil {
						return nil, err
					}
					repositories.ReKyouReps.ReKyouRepositories = append(repositories.ReKyouReps.ReKyouRepositories, reKyouRep)
					repositories.Reps = append(repositories.Reps, reKyouRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteReKyouRep.GetPath(ctx, "")
						newPath, _ := reKyouRep.GetPath(ctx, "")
						if repositories.WriteReKyouRep != nil {
							err := fmt.Errorf("error conflict write reKyou rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteReKyouRep = reKyouRep
					}
				case "directory":
					parentDir := filepath.Join(filename, ".kyou")
					err := os.MkdirAll(parentDir, os.ModePerm)
					if err != nil {
						err = fmt.Errorf("error at make directory %s: %w", parentDir, err)
						return nil, err
					}

					idDBFilename := filepath.Join(parentDir, "id.db")
					idfKyouRep, err := reps.NewIDFDirRep(ctx, filename, idDBFilename, g.router, g.autoIDF, &g.IDFIgnore, repositories)
					if err != nil {
						return nil, err
					}
					repositories.IDFKyouReps = append(repositories.IDFKyouReps, idfKyouRep)
					repositories.Reps = append(repositories.Reps, idfKyouRep)
					if rep.UseToWrite {
						existPath, _ := repositories.WriteIDFKyouRep.GetPath(ctx, "")
						newPath, _ := idfKyouRep.GetPath(ctx, "")
						if repositories.WriteIDFKyouRep != nil {
							err := fmt.Errorf("error conflict write idf kyou rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteIDFKyouRep = idfKyouRep
					}
				}
			}
		}

		g.gkillRepositories[userID][device] = repositories
		repositories, _ = repositoriesInDevice[device]
	}
	return repositories, nil
	// ˄
}

func (g *GkillDAOManager) CloseUserRepositories(userID string, device string) (bool, error) {
	// ˅
	ctx := context.TODO()
	repsInDevices, exist := g.gkillRepositories[userID]
	if !exist {
		return false, nil
	}
	reps, exist := repsInDevices[device]
	if !exist {
		return false, nil
	}
	err := reps.Close(ctx)
	if err != nil {
		fmt.Errorf("error at close repositories: %w", err)
		return false, err
	}
	return true, nil
	// ˄
}

// ˅

// ˄
