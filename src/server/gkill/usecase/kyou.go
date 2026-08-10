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
	kyouHistories := []reps.Kyou{}
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
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetKyouError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	if kyouHistories == nil {
		kyouHistories = []reps.Kyou{}
	}
	return kyouHistories, nil, nil
}

// GetKyous はKyou一覧を取得するユースケース。
// 第2戻り値は致命的ではない警告メッセージ(プラグイン検索の失敗など)で、
// 検索自体は成功として結果と一緒に返す。
func (uc *UsecaseContext) GetKyous(ctx context.Context, userID, device, localeName string, query *find.FindQuery) ([]reps.Kyou, []*message.GkillMessage, []*message.GkillError, error) {
	query.OnlyLatestData = true

	// プラグイン検索の失敗は警告として回収する。
	// エラー(errors)に載せるとハンドラ・クライアントが検索全体を失敗扱いにして
	// 結果を破棄するため、メッセージ(messages)として返す
	ctx = reps.WithFindWarnings(ctx)

	kyous, gkillErrors, err := uc.FindFilter.FindKyous(ctx, userID, device, uc.DAOManager, query)
	if len(gkillErrors) != 0 || err != nil {
		if err != nil {
			err = fmt.Errorf("error at find kyous: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
		return nil, nil, gkillErrors, nil
	}

	var warningMessages []*message.GkillMessage
	for _, pluginName := range reps.PluginFindWarnings(ctx) {
		warningMessages = append(warningMessages, &message.GkillMessage{
			MessageCode: message.FindKyousPluginWarningMessage,
			Message:     api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_FIND_PLUGIN_MESSAGE"}) + " (" + pluginName + ")",
		})
	}

	return kyous, warningMessages, nil, nil
}
