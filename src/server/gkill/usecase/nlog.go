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

// AddNlog はNlogを追加するユースケース
func (uc *UsecaseContext) AddNlog(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, nlog reps.Nlog, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existNlog, err := repositories.NlogReps.GetNlog(ctx, nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existNlog != nil {
		err = fmt.Errorf("exist nlog id = %s", nlog.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistNlogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteNlogRep.AddNlogInfo(ctx, nlog)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.NlogReps) == 1 && *gkill_options.CacheNlogReps {
			err = repositories.NlogReps[0].AddNlogInfo(ctx, nlog)
			if err != nil {
				err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.NlogTempRep.AddNlogInfo(ctx, nlog, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteNlogRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[nlog.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              nlog.IsDeleted,
		TargetID:                               nlog.ID,
		DataUpdateTime:                         nlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[nlog.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for nlog user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	return nil, nil
}

// UpdateNlog はNlogを更新するユースケース
func (uc *UsecaseContext) UpdateNlog(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, nlog reps.Nlog, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	_, err := repositories.NlogReps.GetNlog(ctx, nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteNlogRep.AddNlogInfo(ctx, nlog)
		if err != nil {
			err = fmt.Errorf("error at update nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateNlogError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.NlogReps) == 1 && *gkill_options.CacheNlogReps {
			err = repositories.NlogReps[0].AddNlogInfo(ctx, nlog)
			if err != nil {
				err = fmt.Errorf("error at update nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.NlogTempRep.AddNlogInfo(ctx, nlog, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateNlogError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteNlogRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[nlog.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              nlog.IsDeleted,
		TargetID:                               nlog.ID,
		DataUpdateTime:                         nlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[nlog.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for nlog user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	existNlog, err := repositories.NlogReps.GetNlog(ctx, nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existNlog == nil {
		err = fmt.Errorf("not exist nlog id = %s", nlog.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundNlogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	return nil, nil
}

// GetNlogHistories はNlog履歴を取得するユースケース
func (uc *UsecaseContext) GetNlogHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.Nlog, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	nlogHistories, err := repositories.NlogReps.GetNlogHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NLOG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return nlogHistories, nil, nil
}
