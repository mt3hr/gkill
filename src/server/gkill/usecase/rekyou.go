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

// AddReKyou はReKyouを追加するユースケース
func (uc *UsecaseContext) AddReKyou(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, rekyou reps.ReKyou, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existReKyou, err := repositories.ReKyouReps.GetReKyou(ctx, rekyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existReKyou != nil {
		err = fmt.Errorf("exist rekyou id = %s", rekyou.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteReKyouRep.AddReKyouInfo(ctx, rekyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		err = repositories.WriteThroughReKyouCache(ctx, rekyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
	} else {
		err = repositories.TempReps.ReKyouTempRep.AddReKyouInfo(ctx, rekyou, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteReKyouRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_ADDED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              rekyou.IsDeleted,
		TargetID:                               rekyou.ID,
		TargetIDInData:                         &rekyou.TargetID,
		DataUpdateTime:                         rekyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	repositories.SetLatestDataRepositoryAddress(rekyou.ID, latestDataRepositoryAddress)

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
	}

	return nil, nil
}

// UpdateReKyou はReKyouを更新するユースケース
func (uc *UsecaseContext) UpdateReKyou(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, rekyou reps.ReKyou, txID *string) ([]*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	existReKyou, err := repositories.ReKyouReps.GetReKyou(ctx, rekyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	if existReKyou == nil {
		err = fmt.Errorf("not exist rekyou id = %s", rekyou.ID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
		})
		return gkillErrors, nil
	}

	if txID == nil {
		err = repositories.WriteReKyouRep.AddReKyouInfo(ctx, rekyou)
		if err != nil {
			err = fmt.Errorf("error at update rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
		// **クライアントがエコーした rep_name をそのままキャッシュへ入れない。**
		// 実体は WriteReKyouRep（書き込み先rep）へ足したので、キャッシュ表の REP_NAME もそこに合わせる。
		// 取得元repの名前のまま入れると、端末別にrepを分けている環境で他端末由来の記録を編集したとき、
		// find_filter.go の filterKyousByRepName が「非空で、指定repに無い名前」として落とし、
		// **更新直後だけ一覧から消えて次の UpdateCache（最大1分）で戻る**。
		// 取れなければ空にする ―― 空は filterKyousByRepName が残すので安全側。
		if writeRepName, repNameErr := repositories.WriteReKyouRep.GetRepName(ctx); repNameErr == nil {
			rekyou.RepName = writeRepName
		} else {
			rekyou.RepName = ""
		}
		err = repositories.WriteThroughReKyouCache(ctx, rekyou)
		if err != nil {
			err = fmt.Errorf("error at update rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
	} else {
		err = repositories.TempReps.ReKyouTempRep.AddReKyouInfo(ctx, rekyou, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateReKyouError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
			})
			return gkillErrors, nil
		}
	}

	repName, err := repositories.WriteReKyouRep.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_UPDATED_GET_MESSAGE"}),
		})
		return gkillErrors, nil
	}
	latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              rekyou.IsDeleted,
		TargetID:                               rekyou.ID,
		TargetIDInData:                         &rekyou.TargetID,
		DataUpdateTime:                         rekyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	repositories.SetLatestDataRepositoryAddress(rekyou.ID, latestDataRepositoryAddress)

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
	}

	return nil, nil
}

// GetReKyouHistories はReKyou履歴を取得するユースケース
func (uc *UsecaseContext) GetReKyouHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, repName *string) ([]reps.ReKyou, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	rekyouHistories, err := repositories.ReKyouReps.GetReKyouHistoriesByRepName(ctx, id, repName)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REKYOU_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return rekyouHistories, nil, nil
}

// GetReKyousByTargetID は対象Kyouをリポストしている未削除ReKyouを取得するユースケース。
// 参照先Kyouが削除済みかどうかは見ません（契約は reps.ReKyouRepositories.GetReKyousByTargetID を参照）。
func (uc *UsecaseContext) GetReKyousByTargetID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, targetID string) ([]reps.ReKyou, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	rekyous, err := repositories.GetReKyousByTargetID(ctx, targetID)
	if err != nil {
		err = fmt.Errorf("error at get rekyous by target id user id = %s device = %s target id = %s: %w", userID, device, targetID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetReKyousByTargetIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REKYOU_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return rekyous, nil, nil
}
