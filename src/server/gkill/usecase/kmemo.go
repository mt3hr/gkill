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

// AddKmemo はKmemoを追加するユースケース
func (uc *UsecaseContext) AddKmemo(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, kmemo reps.Kmemo, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// 対象が存在する場合はエラー
	existKmemo, err := repositories.KmemoReps.GetKmemo(ctx, kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existKmemo != nil {
		err = fmt.Errorf("exist kmemo id = %s", kmemo.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistKmemoError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteKmemoRep.AddKmemoInfo(ctx, kmemo)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.KmemoReps) == 1 && *gkill_options.CacheKmemoReps {
			err = repositories.KmemoReps[0].AddKmemoInfo(ctx, kmemo)
			if err != nil {
				err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.KmemoTempRep.AddKmemoInfo(ctx, kmemo, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteKmemoRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[kmemo.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              kmemo.IsDeleted,
		TargetID:                               kmemo.ID,
		DataUpdateTime:                         kmemo.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[kmemo.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for kmemo user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	return nil, nil
}

// UpdateKmemo はKmemoを更新するユースケース
func (uc *UsecaseContext) UpdateKmemo(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, kmemo reps.Kmemo, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// すでに存在する場合はエラー
	_, err := repositories.KmemoReps.GetKmemo(ctx, kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteKmemoRep.AddKmemoInfo(ctx, kmemo)
		if err != nil {
			err = fmt.Errorf("error at update kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateKmemoError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.KmemoReps) == 1 && *gkill_options.CacheKmemoReps {
			err = repositories.KmemoReps[0].AddKmemoInfo(ctx, kmemo)
			if err != nil {
				err = fmt.Errorf("error at update kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.KmemoTempRep.AddKmemoInfo(ctx, kmemo, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateKmemoError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteKmemoRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[kmemo.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              kmemo.IsDeleted,
		TargetID:                               kmemo.ID,
		DataUpdateTime:                         kmemo.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[kmemo.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for kmemo user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	// 対象が存在しない場合はエラー
	existKmemo, err := repositories.KmemoReps.GetKmemo(ctx, kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existKmemo == nil {
		err = fmt.Errorf("not exist kmemo id = %s", kmemo.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundKmemoError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	return nil, nil
}

// GetKmemoHistories はKmemo履歴を取得するユースケース
func (uc *UsecaseContext) GetKmemoHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.Kmemo, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	kmemoHistories, err := repositories.KmemoReps.GetKmemoHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KMEMO_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return kmemoHistories, nil, nil
}
