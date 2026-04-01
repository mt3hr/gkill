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

// AddReKyou はReKyouを追加するユースケース
func (uc *UsecaseContext) AddReKyou(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, rekyou reps.ReKyou, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existReKyou, err := repositories.ReKyouReps.GetReKyou(ctx, rekyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existReKyou != nil {
		err = fmt.Errorf("exist rekyou id = %s", rekyou.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteReKyouRep.AddReKyouInfo(ctx, rekyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.ReKyouReps.ReKyouRepositories) == 1 && *gkill_options.CacheReKyouReps {
			err = repositories.ReKyouReps.ReKyouRepositories[0].AddReKyouInfo(ctx, rekyou)
			if err != nil {
				err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.ReKyouTempRep.AddReKyouInfo(ctx, rekyou, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteReKyouRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[rekyou.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              rekyou.IsDeleted,
		TargetID:                               rekyou.ID,
		TargetIDInData:                         &rekyou.TargetID,
		DataUpdateTime:                         rekyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[rekyou.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	return nil, nil
}

// UpdateReKyou はReKyouを更新するユースケース
func (uc *UsecaseContext) UpdateReKyou(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, rekyou reps.ReKyou, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	_, err := repositories.ReKyouReps.GetReKyou(ctx, rekyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteReKyouRep.AddReKyouInfo(ctx, rekyou)
		if err != nil {
			err = fmt.Errorf("error at update rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.ReKyouReps.ReKyouRepositories) == 1 && *gkill_options.CacheReKyouReps {
			err = repositories.ReKyouReps.ReKyouRepositories[0].AddReKyouInfo(ctx, rekyou)
			if err != nil {
				err = fmt.Errorf("error at update rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.ReKyouTempRep.AddReKyouInfo(ctx, rekyou, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteReKyouRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[rekyou.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              rekyou.IsDeleted,
		TargetID:                               rekyou.ID,
		TargetIDInData:                         &rekyou.TargetID,
		DataUpdateTime:                         rekyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[rekyou.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	existReKyou, err := repositories.ReKyouReps.GetReKyou(ctx, rekyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existReKyou == nil {
		err = fmt.Errorf("not exist rekyou id = %s", rekyou.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	return nil, nil
}

// GetReKyouHistories はReKyou履歴を取得するユースケース
func (uc *UsecaseContext) GetReKyouHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.ReKyou, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	rekyouHistories, err := repositories.ReKyouReps.GetReKyouHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REKYOU_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return rekyouHistories, nil, nil
}
