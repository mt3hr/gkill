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

// AddTimeIs はTimeIsを追加するユースケース
func (uc *UsecaseContext) AddTimeIs(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, timeis reps.TimeIs, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existTimeIs, err := repositories.TimeIsReps.GetTimeIs(ctx, timeis.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existTimeIs != nil {
		err = fmt.Errorf("exist timeis id = %s", timeis.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(ctx, timeis)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
			err = repositories.TimeIsReps[0].AddTimeIsInfo(ctx, timeis)
			if err != nil {
				err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(ctx, timeis, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteTimeIsRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[timeis.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              timeis.IsDeleted,
		TargetID:                               timeis.ID,
		DataUpdateTime:                         timeis.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[timeis.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	return nil, nil
}

// UpdateTimeIs はTimeIsを更新するユースケース
func (uc *UsecaseContext) UpdateTimeIs(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, timeis reps.TimeIs, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	_, err := repositories.TimeIsReps.GetTimeIs(ctx, timeis.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(ctx, timeis)
		if err != nil {
			err = fmt.Errorf("error at update timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateTimeIsError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
			err = repositories.TimeIsReps[0].AddTimeIsInfo(ctx, timeis)
			if err != nil {
				err = fmt.Errorf("error at update timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(ctx, timeis, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateTimeIsError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteTimeIsRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[timeis.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              timeis.IsDeleted,
		TargetID:                               timeis.ID,
		DataUpdateTime:                         timeis.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[timeis.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	existTimeIs, err := repositories.TimeIsReps.GetTimeIs(ctx, timeis.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existTimeIs == nil {
		err = fmt.Errorf("not exist timeis id = %s", timeis.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	return nil, nil
}

// GetTimeIsHistories はTimeIs履歴を取得するユースケース
func (uc *UsecaseContext) GetTimeIsHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.TimeIs, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	timeisHistories, err := repositories.TimeIsReps.GetTimeIsHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TIMEIS_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return timeisHistories, nil, nil
}
