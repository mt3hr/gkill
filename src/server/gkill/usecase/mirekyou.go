package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// AddMiReKyou はMiReKyouを追加するユースケース
func (uc *UsecaseContext) AddMiReKyou(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, mirekyou reps.MiReKyou, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existMiReKyou, err := repositories.MiReKyouReps.GetMiReKyou(ctx, mirekyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mirekyou user id = %s device = %s id = %s: %w", userID, device, mirekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existMiReKyou != nil {
		err = fmt.Errorf("exist mirekyou id = %s", mirekyou.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistMiReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		if repositories.WriteMiReKyouRep == nil {
			err = fmt.Errorf("not exist write mirekyou rep user id = %s device = %s", userID, device)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddMiReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		err = repositories.WriteMiReKyouRep.AddMiReKyouInfo(ctx, mirekyou)
		if err != nil {
			err = fmt.Errorf("error at add mirekyou user id = %s device = %s mirekyou = %#v: %w", userID, device, mirekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddMiReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.MiReKyouReps.MiReKyouRepositories) == 1 && *gkill_options.CacheMiReKyouReps {
			err = repositories.MiReKyouReps.MiReKyouRepositories[0].AddMiReKyouInfo(ctx, mirekyou)
			if err != nil {
				err = fmt.Errorf("error at add mirekyou user id = %s device = %s mirekyou = %#v: %w", userID, device, mirekyou, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			}
		}
	} else {
		err = repositories.TempReps.MiReKyouTempRep.AddMiReKyouInfo(ctx, mirekyou, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add mirekyou user id = %s device = %s mirekyou = %#v: %w", userID, device, mirekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddMiReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		return nil, nil
	}

	gkillErrors = uc.updateMiReKyouLatestDataRepositoryAddress(ctx, repositories, userID, device, localeName, mirekyou)
	if len(gkillErrors) > 0 {
		return gkillErrors, nil
	}

	return nil, nil
}

// UpdateMiReKyou はMiReKyouを更新するユースケース
func (uc *UsecaseContext) UpdateMiReKyou(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, mirekyou reps.MiReKyou, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// 対象が存在しない場合はエラー
	existMiReKyou, err := repositories.MiReKyouReps.GetMiReKyou(ctx, mirekyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mirekyou user id = %s device = %s id = %s: %w", userID, device, mirekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existMiReKyou == nil {
		err = fmt.Errorf("not exist mirekyou id = %s", mirekyou.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundMiReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		if repositories.WriteMiReKyouRep == nil {
			err = fmt.Errorf("not exist write mirekyou rep user id = %s device = %s", userID, device)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateMiReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		err = repositories.WriteMiReKyouRep.AddMiReKyouInfo(ctx, mirekyou)
		if err != nil {
			err = fmt.Errorf("error at update mirekyou user id = %s device = %s mirekyou = %#v: %w", userID, device, mirekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateMiReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.MiReKyouReps.MiReKyouRepositories) == 1 && *gkill_options.CacheMiReKyouReps {
			err = repositories.MiReKyouReps.MiReKyouRepositories[0].AddMiReKyouInfo(ctx, mirekyou)
			if err != nil {
				err = fmt.Errorf("error at update mirekyou user id = %s device = %s mirekyou = %#v: %w", userID, device, mirekyou, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			}
		}
	} else {
		err = repositories.TempReps.MiReKyouTempRep.AddMiReKyouInfo(ctx, mirekyou, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update mirekyou user id = %s device = %s mirekyou = %#v: %w", userID, device, mirekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateMiReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		return nil, nil
	}

	gkillErrors = uc.updateMiReKyouLatestDataRepositoryAddress(ctx, repositories, userID, device, localeName, mirekyou)
	if len(gkillErrors) > 0 {
		return gkillErrors, nil
	}

	return nil, nil
}

// updateMiReKyouLatestDataRepositoryAddress は最新データ位置キャッシュを更新します。
// ReKyouと同じくTargetIDInDataにリポスト対象のIDを入れます。
func (uc *UsecaseContext) updateMiReKyouLatestDataRepositoryAddress(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, mirekyou reps.MiReKyou) []*message.GkillError {
	var gkillErrors []*message.GkillError

	repName, err := repositories.WriteMiReKyouRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, mirekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors
	}
	latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              mirekyou.IsDeleted,
		TargetID:                               mirekyou.ID,
		TargetIDInData:                         &mirekyou.TargetID,
		DataUpdateTime:                         mirekyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	repositories.SetLatestDataRepositoryAddress(mirekyou.ID, latestDataRepositoryAddress)

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for mirekyou user id = %s device = %s id = %s: %w", userID, device, mirekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
	}
	return nil
}

// GetMiReKyouHistories はMiReKyou履歴を取得するユースケース
func (uc *UsecaseContext) GetMiReKyouHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.MiReKyou, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	mirekyouHistories, err := repositories.MiReKyouReps.GetMiReKyouHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get mirekyou user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_REKYOU_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return mirekyouHistories, nil, nil
}

// GetMiReKyousByTargetID は対象Kyouをタスク化している未削除MiReKyouを取得するユースケース。
// 参照先Kyouが削除済みかどうかは見ません（契約は reps.MiReKyouRepositories.GetMiReKyousByTargetID を参照）。
func (uc *UsecaseContext) GetMiReKyousByTargetID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, targetID string) ([]reps.MiReKyou, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	mirekyous, err := repositories.GetMiReKyousByTargetID(ctx, targetID)
	if err != nil {
		err = fmt.Errorf("error at get mirekyous by target id user id = %s device = %s target id = %s: %w", userID, device, targetID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiReKyousByTargetIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_REKYOU_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return mirekyous, nil, nil
}
