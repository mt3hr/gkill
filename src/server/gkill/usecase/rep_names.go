package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// GetAllTagNames は全タグ名を取得するユースケース
func (uc *UsecaseContext) GetAllTagNames(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string) ([]string, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	allTagNames, err := repositories.GetAllTagNames(context.Background())
	if err != nil {
		err = fmt.Errorf("error at get all tag names user id = %s device = %s: %w", userID, device, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetAllTagNamesError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_TAG_NAMES_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return allTagNames, nil, nil
}

// GetAllRepNames は全レポジトリ名を取得するユースケース
func (uc *UsecaseContext) GetAllRepNames(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string) ([]string, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	allRepNames, err := repositories.GetAllRepNames(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all rep names user id = %s device = %s: %w", userID, device, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetAllRepNamesError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_REP_NAMES_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return allRepNames, nil, nil
}
