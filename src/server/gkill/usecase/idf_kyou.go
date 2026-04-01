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

// UpdateIDFKyou はIDFKyouを更新するユースケース
func (uc *UsecaseContext) UpdateIDFKyou(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, idfKyou reps.IDFKyou, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	if txID == nil {
		err := repositories.WriteIDFKyouRep.AddIDFKyouInfo(ctx, idfKyou)
		if err != nil {
			err = fmt.Errorf("error at update idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, idfKyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateIDFKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.IDFKyouReps) == 1 && *gkill_options.CacheIDFKyouReps {
			err = repositories.IDFKyouReps[0].AddIDFKyouInfo(ctx, idfKyou)
			if err != nil {
				err = fmt.Errorf("error at update idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, idfKyou, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err := repositories.TempReps.IDFKyouTempRep.AddIDFKyouInfo(ctx, idfKyou, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, idfKyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateIDFKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteIDFKyouRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, idfKyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[idfKyou.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              idfKyou.IsDeleted,
		TargetID:                               idfKyou.ID,
		DataUpdateTime:                         idfKyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[idfKyou.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for idfKyou user id = %s device = %s id = %s: %w", userID, device, idfKyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	existIDFKyou, err := repositories.IDFKyouReps.GetIDFKyou(ctx, idfKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, idfKyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existIDFKyou == nil {
		err = fmt.Errorf("not exist idfKyou id = %s", idfKyou.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundIDFKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	return nil, nil
}

// GetIDFKyouHistories はIDFKyou履歴を取得するユースケース
func (uc *UsecaseContext) GetIDFKyouHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.IDFKyou, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	idfKyouHistories, err := repositories.IDFKyouReps.GetIDFKyouHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_IDFKYOU_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return idfKyouHistories, nil, nil
}
