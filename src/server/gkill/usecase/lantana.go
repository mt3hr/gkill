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

// AddLantana はLantanaを追加するユースケース
func (uc *UsecaseContext) AddLantana(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, lantana reps.Lantana, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existLantana, err := repositories.LantanaReps.GetLantana(ctx, lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existLantana != nil {
		err = fmt.Errorf("exist lantana id = %s", lantana.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistLantanaError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteLantanaRep.AddLantanaInfo(ctx, lantana)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.LantanaReps) == 1 && *gkill_options.CacheLantanaReps {
			err = repositories.LantanaReps[0].AddLantanaInfo(ctx, lantana)
			if err != nil {
				err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.LantanaTempRep.AddLantanaInfo(ctx, lantana, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteLantanaRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[lantana.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              lantana.IsDeleted,
		TargetID:                               lantana.ID,
		DataUpdateTime:                         lantana.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[lantana.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for lantana user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	return nil, nil
}

// UpdateLantana はLantanaを更新するユースケース
func (uc *UsecaseContext) UpdateLantana(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, lantana reps.Lantana, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	if txID == nil {
		err := repositories.WriteLantanaRep.AddLantanaInfo(ctx, lantana)
		if err != nil {
			err = fmt.Errorf("error at update lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateLantanaError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		if len(repositories.LantanaReps) == 1 && *gkill_options.CacheLantanaReps {
			err = repositories.LantanaReps[0].AddLantanaInfo(ctx, lantana)
			if err != nil {
				err = fmt.Errorf("error at update lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err := repositories.TempReps.LantanaTempRep.AddLantanaInfo(ctx, lantana, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateLantanaError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteLantanaRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[lantana.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              lantana.IsDeleted,
		TargetID:                               lantana.ID,
		DataUpdateTime:                         lantana.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[lantana.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for lantana user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	existLantana, err := repositories.LantanaReps.GetLantana(ctx, lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existLantana == nil {
		err = fmt.Errorf("not exist lantana id = %s", lantana.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundLantanaError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	return nil, nil
}

// GetLantanaHistories はLantana履歴を取得するユースケース
func (uc *UsecaseContext) GetLantanaHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.Lantana, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	lantanaHistories, err := repositories.LantanaReps.GetLantanaHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_LANTANA_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return lantanaHistories, nil, nil
}
