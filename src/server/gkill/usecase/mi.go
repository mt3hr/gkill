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

// AddMi はMiを追加するユースケース
func (uc *UsecaseContext) AddMi(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, mi reps.Mi, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existMi, err := repositories.MiReps.GetMi(ctx, mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existMi != nil {
		err = fmt.Errorf("exist mi id = %s", mi.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistMiError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteMiRep.AddMiInfo(ctx, mi)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.MiReps) == 1 && *gkill_options.CacheMiReps {
			err = repositories.MiReps[0].AddMiInfo(ctx, mi)
			if err != nil {
				err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.MiTempRep.AddMiInfo(ctx, mi, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteMiRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[mi.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              mi.IsDeleted,
		TargetID:                               mi.ID,
		DataUpdateTime:                         mi.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[mi.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for mi user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	return nil, nil
}

// UpdateMi はMiを更新するユースケース
func (uc *UsecaseContext) UpdateMi(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, mi reps.Mi, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// 対象が存在しない場合はエラー
	existMi, err := repositories.MiReps.GetMi(ctx, mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existMi == nil {
		err = fmt.Errorf("not exist mi id = %s", mi.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundMiError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteMiRep.AddMiInfo(ctx, mi)
		if err != nil {
			err = fmt.Errorf("error at update mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateMiError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.MiReps) == 1 && *gkill_options.CacheMiReps {
			err = repositories.MiReps[0].AddMiInfo(ctx, mi)
			if err != nil {
				err = fmt.Errorf("error at update mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.MiTempRep.AddMiInfo(ctx, mi, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateMiError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteMiRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[mi.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              mi.IsDeleted,
		TargetID:                               mi.ID,
		DataUpdateTime:                         mi.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[mi.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for mi user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	return nil, nil
}

// GetMiHistories はMi履歴を取得するユースケース
func (uc *UsecaseContext) GetMiHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.Mi, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	miHistories, err := repositories.MiReps.GetMiHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return miHistories, nil, nil
}

// GetMiBoardList はMiボードリストを取得するユースケース
func (uc *UsecaseContext) GetMiBoardList(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string) ([]string, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	miBoardNames, err := repositories.MiReps.GetBoardNames(ctx)
	if err != nil {
		err = fmt.Errorf("error at get mi board names user id = %s device = %s: %w", userID, device, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetMiBoardNamesError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return miBoardNames, nil, nil
}
