// ˅
package req_res

// ˄

type GetGitCommitLogResponse struct {
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
