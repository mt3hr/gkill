// ˅
package req_res

// ˄

type GetGitCommitLogsResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	GitCommitLogs []*GitCommitLog `json:"git_commit_logs"`

	// ˅

	// ˄
}

// ˅

// ˄
