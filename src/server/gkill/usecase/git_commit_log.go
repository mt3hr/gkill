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

// GetGitCommitLog はGitCommitLogを取得するユースケース
func (uc *UsecaseContext) GetGitCommitLog(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string) ([]reps.GitCommitLog, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	gitCommitLog, err := repositories.GitCommitLogReps.GetGitCommitLog(ctx, id, nil)
	if err != nil {
		err = fmt.Errorf("error at get gitCommitLog user id = %s device = %s id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetGitCommitLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GIT_COMMIT_LOG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}
	if gitCommitLog == nil {
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetGitCommitLogError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GIT_COMMIT_LOG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	return []reps.GitCommitLog{*gitCommitLog}, nil, nil
}
