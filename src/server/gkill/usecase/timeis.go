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
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// AddTimeIs はTimeIsを追加するユースケース
func (uc *UsecaseContext) AddTimeIs(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, timeis reps.TimeIs, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existTimeIs, err := repositories.TimeIsReps.GetTimeIs(ctx, timeis.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existTimeIs != nil {
		err = fmt.Errorf("exist timeis id = %s", timeis.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		err = repositories.WriteThroughTimeIsCache(ctx, timeis)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
	} else {
		err = repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(ctx, timeis, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              timeis.IsDeleted,
		TargetID:                               timeis.ID,
		DataUpdateTime:                         timeis.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	repositories.SetLatestDataRepositoryAddress(timeis.ID, latestDataRepositoryAddress)

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
	}

	return nil, nil
}

// UpdateTimeIs はTimeIsを更新するユースケース
func (uc *UsecaseContext) UpdateTimeIs(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, timeis reps.TimeIs, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existTimeIs, err := repositories.TimeIsReps.GetTimeIs(ctx, timeis.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existTimeIs == nil {
		err = fmt.Errorf("not exist timeis id = %s", timeis.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(ctx, timeis)
		if err != nil {
			err = fmt.Errorf("error at update timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateTimeIsError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		// **クライアントがエコーした rep_name をそのままキャッシュへ入れない。**
		// 実体は WriteTimeIsRep（書き込み先rep）へ足したので、キャッシュ表の REP_NAME もそこに合わせる。
		// 取得元repの名前のまま入れると、端末別にrepを分けている環境で他端末由来の記録を編集したとき、
		// find_filter.go の filterKyousByRepName が「非空で、指定repに無い名前」として落とし、
		// **更新直後だけ一覧から消えて次の UpdateCache（最大1分）で戻る**。
		// 取れなければ空にする ―― 空は filterKyousByRepName が残すので安全側。
		if writeRepName, repNameErr := repositories.WriteTimeIsRep.GetRepName(ctx); repNameErr == nil {
			timeis.RepName = writeRepName
		} else {
			timeis.RepName = ""
		}
		err = repositories.WriteThroughTimeIsCache(ctx, timeis)
		if err != nil {
			err = fmt.Errorf("error at update timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
	} else {
		err = repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(ctx, timeis, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              timeis.IsDeleted,
		TargetID:                               timeis.ID,
		DataUpdateTime:                         timeis.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	repositories.SetLatestDataRepositoryAddress(timeis.ID, latestDataRepositoryAddress)

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
	}

	return nil, nil
}

// GetTimeIsHistories はTimeIs履歴を取得するユースケース
func (uc *UsecaseContext) GetTimeIsHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.TimeIs, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	timeisHistories, err := repositories.TimeIsReps.GetTimeIsHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TIMEIS_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return timeisHistories, nil, nil
}
