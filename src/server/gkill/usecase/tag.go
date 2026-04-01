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

// AddTag はタグを追加するユースケース
func (uc *UsecaseContext) AddTag(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, tag reps.Tag, txID *string) (*reps.Tag, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// 対象が存在する場合はエラー
	existTag, err := repositories.GetTag(ctx, tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	if existTag != nil {
		err = fmt.Errorf("exist tag id = %s", tag.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteTagRep.AddTagInfo(ctx, tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
		// キャッシュに書き込み
		if len(repositories.TagReps) == 1 && *gkill_options.CacheTagReps {
			err = repositories.TagReps[0].AddTagInfo(ctx, tag)
			if err != nil {
				err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TagTempRep.AddTagInfo(ctx, tag, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
	}

	repName, err := repositories.WriteTagRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_ADDED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[tag.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              tag.IsDeleted,
		TargetID:                               tag.ID,
		TargetIDInData:                         &tag.TargetID,
		DataUpdateTime:                         tag.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[tag.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	addedTag, err := repositories.GetTag(ctx, tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_ADDED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return addedTag, nil, nil
}

// UpdateTag はタグを更新するユースケース
func (uc *UsecaseContext) UpdateTag(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, tag reps.Tag, txID *string) (*reps.Tag, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// すでに存在する場合はエラー
	_, err := repositories.GetTag(ctx, tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_UPDATED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteTagRep.AddTagInfo(ctx, tag)
		if err != nil {
			err = fmt.Errorf("error at update tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}

		// キャッシュに書き込み
		if len(repositories.TagReps) == 1 && *gkill_options.CacheTagReps {
			err = repositories.TagReps[0].AddTagInfo(ctx, tag)
			if err != nil {
				err = fmt.Errorf("error at update tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TagTempRep.AddTagInfo(ctx, tag, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
	}

	repName, err := repositories.WriteTagRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_UPDATED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[tag.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              tag.IsDeleted,
		TargetID:                               tag.ID,
		TargetIDInData:                         &tag.TargetID,
		DataUpdateTime:                         tag.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[tag.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	updatedTag, err := repositories.GetTag(ctx, tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_UPDATED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	// 対象が存在しない場合はエラー
	existTag, err := repositories.GetTag(ctx, tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	if existTag == nil {
		err = fmt.Errorf("not exist tag id = %s", tag.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return updatedTag, nil, nil
}

// GetTagsByTargetID はターゲットIDに紐づくタグを取得するユースケース
func (uc *UsecaseContext) GetTagsByTargetID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, targetID string) ([]reps.Tag, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	tags, err := repositories.GetTagsByTargetID(ctx, targetID)
	if err != nil {
		err = fmt.Errorf("error at get tags by target id user id = %s device = %s target id = %s: %w", userID, device, targetID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagsByTargetIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REPOSITORIES_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return tags, nil, nil
}

// GetTagHistoriesByTagID はタグIDに紐づくタグ履歴を取得するユースケース
func (uc *UsecaseContext) GetTagHistoriesByTagID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, updateTime *time.Time, repName *string) ([]reps.Tag, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	var tags []reps.Tag
	var err error
	if updateTime != nil {
		var tag *reps.Tag
		tag, err = repositories.GetTag(ctx, id, updateTime)
		if err == nil && tag != nil {
			tags = []reps.Tag{*tag}
		}
	} else {
		tags, err = repositories.TagReps.GetTagHistoriesByRepName(ctx, id, repName)
	}

	if err != nil {
		err = fmt.Errorf("error at get tag histories by tag id user id = %s device = %s target id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagHistoriesByTagIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAG_HISTORIES_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return tags, nil, nil
}
