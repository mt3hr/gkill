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

// AddNotification は通知を追加するユースケース（通知情報更新はハンドラ側で実行）
func (uc *UsecaseContext) AddNotification(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, notification reps.Notification, txID *string) (*reps.Notification, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// 対象が存在する場合はエラー
	existNotification, err := repositories.GetNotification(ctx, notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	if existNotification != nil {
		err = fmt.Errorf("exist notification id = %s", notification.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistNotificationError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteNotificationRep.AddNotificationInfo(ctx, notification)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
		if len(repositories.NotificationReps) == 1 && *gkill_options.CacheNotificationReps {
			err = repositories.NotificationReps[0].AddNotificationInfo(ctx, notification)
			if err != nil {
				err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.NotificationTempRep.AddNotificationInfo(ctx, notification, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
	}

	repName, err := repositories.WriteNotificationRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_ADDED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[notification.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              notification.IsDeleted,
		TargetID:                               notification.ID,
		TargetIDInData:                         &notification.TargetID,
		DataUpdateTime:                         notification.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[notification.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	addedNotification, err := repositories.GetNotification(ctx, notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_ADDED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return addedNotification, nil, nil
}

// UpdateNotification は通知を更新するユースケース（通知情報更新はハンドラ側で実行）
func (uc *UsecaseContext) UpdateNotification(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, notification reps.Notification, txID *string) (*reps.Notification, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// すでに存在する場合はエラー
	_, err := repositories.GetNotification(ctx, notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_UPDATED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteNotificationRep.AddNotificationInfo(ctx, notification)
		if err != nil {
			err = fmt.Errorf("error at update notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateNotificationError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
		if len(repositories.NotificationReps) == 1 && *gkill_options.CacheNotificationReps {
			err = repositories.NotificationReps[0].AddNotificationInfo(ctx, notification)
			if err != nil {
				err = fmt.Errorf("error at update notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.NotificationTempRep.AddNotificationInfo(ctx, notification, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateNotificationError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
	}

	repName, err := repositories.WriteNotificationRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_UPDATED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	repositories.LatestDataRepositoryAddresses[notification.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              notification.IsDeleted,
		TargetID:                               notification.ID,
		TargetIDInData:                         &notification.TargetID,
		DataUpdateTime:                         notification.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, repositories.LatestDataRepositoryAddresses[notification.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	updatedNotification, err := repositories.GetNotification(ctx, notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_UPDATED_GET_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	// 対象が存在しない場合はエラー
	existNotification, err := repositories.GetNotification(ctx, notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	if existNotification == nil {
		err = fmt.Errorf("not exist notification id = %s", notification.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundNotificationError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return updatedNotification, nil, nil
}

// GetNotificationsByTargetID はターゲットIDに紐づく通知を取得するユースケース
func (uc *UsecaseContext) GetNotificationsByTargetID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, targetID string) ([]reps.Notification, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	notifications, err := repositories.GetNotificationsByTargetID(ctx, targetID)
	if err != nil {
		err = fmt.Errorf("error at get notifications by target id user id = %s device = %s target id = %s: %w", userID, device, targetID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNotificationsByTargetIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return notifications, nil, nil
}

// GetNotificationHistoriesByNotificationID は通知IDに紐づく通知履歴を取得するユースケース
func (uc *UsecaseContext) GetNotificationHistoriesByNotificationID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, updateTime *time.Time) ([]reps.Notification, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	var notifications []reps.Notification
	var err error
	if updateTime != nil {
		var notification *reps.Notification
		notification, err = repositories.GetNotification(ctx, id, updateTime)
		if err == nil && notification != nil {
			notifications = []reps.Notification{*notification}
		}
	} else {
		notifications, err = repositories.GetNotificationHistories(ctx, id)
	}

	if err != nil {
		err = fmt.Errorf("error at get notification histories by notification id user id = %s device = %s target id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetNotificationHistoriesByNotificationIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return notifications, nil, nil
}
