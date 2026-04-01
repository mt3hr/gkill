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

// AddText はテキストを追加するユースケース
func (uc *UsecaseContext) AddText(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, text reps.Text, txID *string) (*reps.Text, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// 対象が存在する場合はエラー
	existText, err := repositories.GetText(ctx, text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	if existText != nil {
		err = fmt.Errorf("exist text id = %s", text.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistTextError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteTextRep.AddTextInfo(ctx, text)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, text, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
		if len(repositories.TextReps) == 1 && *gkill_options.CacheTextReps {
			err = repositories.TextReps[0].AddTextInfo(ctx, text)
			if err != nil {
				err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, text, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TextTempRep.AddTextInfo(ctx, text, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, text, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
	}

	repName, err := repositories.WriteTextRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_ADDED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[text.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              text.IsDeleted,
		TargetID:                               text.ID,
		TargetIDInData:                         &text.TargetID,
		DataUpdateTime:                         text.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[text.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	addedText, err := repositories.GetText(ctx, text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_ADDED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return addedText, nil, nil
}

// UpdateText はテキストを更新するユースケース
func (uc *UsecaseContext) UpdateText(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, text reps.Text, txID *string) (*reps.Text, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// すでに存在する場合はエラー
	_, err := repositories.GetText(ctx, text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_UPDATED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteTextRep.AddTextInfo(ctx, text)
		if err != nil {
			err = fmt.Errorf("error at update text user id = %s device = %s text = %#v: %w", userID, device, text, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateTextError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}

		if len(repositories.TextReps) == 1 && *gkill_options.CacheTextReps {
			err = repositories.TextReps[0].AddTextInfo(ctx, text)
			if err != nil {
				err = fmt.Errorf("error at update text user id = %s device = %s text = %#v: %w", userID, device, text, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TextTempRep.AddTextInfo(ctx, text, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update text user id = %s device = %s text = %#v: %w", userID, device, text, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateTextError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
	}

	repName, err := repositories.WriteTextRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_UPDATED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[text.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              text.IsDeleted,
		TargetID:                               text.ID,
		TargetIDInData:                         &text.TargetID,
		DataUpdateTime:                         text.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[text.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	updatedText, err := repositories.GetText(ctx, text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_UPDATED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	// 対象が存在しない場合はエラー
	existText, err := repositories.GetText(ctx, text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	if existText == nil {
		err = fmt.Errorf("not exist text id = %s", text.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundTextError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return updatedText, nil, nil
}

// GetTextsByTargetID はターゲットIDに紐づくテキストを取得するユースケース
func (uc *UsecaseContext) GetTextsByTargetID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, targetID string) ([]reps.Text, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	texts, err := repositories.GetTextsByTargetID(ctx, targetID)
	if err != nil {
		err = fmt.Errorf("error at get texts by target id user id = %s device = %s target id = %s: %w", userID, device, targetID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTextsByTargetIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return texts, nil, nil
}

// GetTextHistoriesByTextID はテキストIDに紐づくテキスト履歴を取得するユースケース
func (uc *UsecaseContext) GetTextHistoriesByTextID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, updateTime *time.Time, repName *string) ([]reps.Text, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	var texts []reps.Text
	var err error
	if updateTime != nil {
		var text *reps.Text
		text, err = repositories.GetText(ctx, id, updateTime)
		if err == nil && text != nil {
			texts = []reps.Text{*text}
		}
	} else {
		texts, err = repositories.TextReps.GetTextHistoriesByRepName(ctx, id, repName)
	}

	if err != nil {
		err = fmt.Errorf("error at get text histories by text id user id = %s device = %s target id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTextHistoriesByTextIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return texts, nil, nil
}
