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

// AddURLog はURLogを追加するユースケース（FillURLogFieldはハンドラ側で実行済みの前提）
func (uc *UsecaseContext) AddURLog(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, urlog reps.URLog, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existURLog, err := repositories.URLogReps.GetURLog(ctx, urlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existURLog != nil {
		err = fmt.Errorf("exist urlog id = %s", urlog.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistURLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteURLogRep.AddURLogInfo(ctx, urlog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.URLogReps) == 1 && *gkill_options.CacheURLogReps {
			err = repositories.URLogReps[0].AddURLogInfo(ctx, urlog)
			if err != nil {
				err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.URLogTempRep.AddURLogInfo(ctx, urlog, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteURLogRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[urlog.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              urlog.IsDeleted,
		TargetID:                               urlog.ID,
		DataUpdateTime:                         urlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[urlog.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	return nil, nil
}

// UpdateURLog はURLogを更新するユースケース（FillURLogField/ReGetURLogContentはハンドラ側で実行済みの前提）
func (uc *UsecaseContext) UpdateURLog(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, urlog reps.URLog, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	_, err := repositories.URLogReps.GetURLog(ctx, urlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteURLogRep.AddURLogInfo(ctx, urlog)
		if err != nil {
			err = fmt.Errorf("error at update urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateURLogError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.URLogReps) == 1 && *gkill_options.CacheURLogReps {
			err = repositories.URLogReps[0].AddURLogInfo(ctx, urlog)
			if err != nil {
				err = fmt.Errorf("error at update urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.URLogTempRep.AddURLogInfo(ctx, urlog, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateURLogError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteURLogRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[urlog.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              urlog.IsDeleted,
		TargetID:                               urlog.ID,
		DataUpdateTime:                         urlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[urlog.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	existURLog, err := repositories.URLogReps.GetURLog(ctx, urlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existURLog == nil {
		err = fmt.Errorf("not exist urlog id = %s", urlog.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundURLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	return nil, nil
}

// GetURLogHistories はURLog履歴を取得するユースケース
func (uc *UsecaseContext) GetURLogHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.URLog, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	urlogHistories, err := repositories.URLogReps.GetURLogHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_URLOG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return urlogHistories, nil, nil
}
