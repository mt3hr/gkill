// ˅
package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

// ˄

type GetGitCommitLogsResponse struct {
	// ˅

	// ˄

	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	GitCommitLogs []*reps.GitCommitLog `json:"git_commit_logs"`

	// ˅

	// ˄
}

// ˅

// ˄
