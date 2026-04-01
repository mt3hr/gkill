package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// GetKyouHistories はKyou履歴を取得するユースケース
func (uc *UsecaseContext) GetKyouHistories(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, updateTime *time.Time, repName *string) ([]reps.Kyou, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	var kyouHistories []reps.Kyou
	var err error
	if updateTime != nil {
		var kyou *reps.Kyou
		kyou, err = repositories.GetKyou(ctx, id, updateTime)
		if err == nil && kyou != nil {
			kyouHistories = []reps.Kyou{*kyou}
		}
	} else {
		kyouHistories, err = repositories.Reps.GetKyouHistoriesByRepName(ctx, id, repName)
	}

	if err != nil {
		err = fmt.Errorf("error at get kyou user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return kyouHistories, nil, nil
}

// GetKyous はKyou一覧を取得するユースケース
func (uc *UsecaseContext) GetKyous(ctx context.Context, userID, device, localeName string, query *find.FindQuery) ([]reps.Kyou, []*message.GkillError, error) {
	query.OnlyLatestData = true

	kyous, gkillErrors, err := uc.FindFilter.FindKyous(ctx, userID, device, uc.DAOManager, query)
	if len(gkillErrors) != 0 || err != nil {
		if err != nil {
			err = fmt.Errorf("error at find kyous: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		}
		return nil, gkillErrors, nil
	}

	return kyous, nil, nil
}
