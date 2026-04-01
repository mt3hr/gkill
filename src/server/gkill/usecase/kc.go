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

// AddKC はKCを追加するユースケース
func (uc *UsecaseContext) AddKC(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, kc reps.KC, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existKC, err := repositories.KCReps.GetKC(ctx, kc.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existKC != nil {
		err = fmt.Errorf("exist kc id = %s", kc.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistKCError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteKCRep.AddKCInfo(ctx, kc)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.KCReps) == 1 && *gkill_options.CacheKCReps {
			err = repositories.KCReps[0].AddKCInfo(ctx, kc)
			if err != nil {
				err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.KCTempRep.AddKCInfo(ctx, kc, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteKCRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[kc.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              kc.IsDeleted,
		TargetID:                               kc.ID,
		DataUpdateTime:                         kc.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[kc.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for kc user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	return nil, nil
}

// UpdateKC はKCを更新するユースケース
func (uc *UsecaseContext) UpdateKC(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, kc reps.KC, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	_, err := repositories.KCReps.GetKC(ctx, kc.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteKCRep.AddKCInfo(ctx, kc)
		if err != nil {
			err = fmt.Errorf("error at update kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateKCError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.KCReps) == 1 && *gkill_options.CacheKCReps {
			err = repositories.WriteKCRep.AddKCInfo(ctx, kc)
			if err != nil {
				err = fmt.Errorf("error at update kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.KCTempRep.AddKCInfo(ctx, kc, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateKCError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteKCRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[kc.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              kc.IsDeleted,
		TargetID:                               kc.ID,
		DataUpdateTime:                         kc.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[kc.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for kc user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	existKC, err := repositories.KCReps.GetKC(ctx, kc.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existKC == nil {
		err = fmt.Errorf("not exist kc id = %s", kc.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundKCError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	return nil, nil
}

// GetKCHistories はKC履歴を取得するユースケース
func (uc *UsecaseContext) GetKCHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.KC, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	kcHistories, err := repositories.KCReps.GetKCHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KC_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return kcHistories, nil, nil
}
