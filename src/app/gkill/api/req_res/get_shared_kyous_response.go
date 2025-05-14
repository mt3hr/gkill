package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type GetSharedKyousResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	Title string `json:"title"`

	ViewType string `json:"view_type"`

	Kyous []*reps.Kyou `json:"kyous"`

	Kmemos []*reps.Kmemo `json:"kmemos"`

	KCs []*reps.KC `json:"kcs"`

	Mis []*reps.Mi `json:"mis"`

	Nlogs []*reps.Nlog `json:"nlogs"`

	Lantanas []*reps.Lantana `json:"lantanas"`

	URLogs []*reps.URLog `json:"urlogs"`

	IDFKyous []*reps.IDFKyou `json:"idf_kyous"`

	ReKyous []*reps.ReKyou `json:"rekyous"`

	GitCommitLogs []*reps.GitCommitLog `json:"git_commit_logs"`

	GPSLogs []*reps.GPSLog `json:"gps_logs"`

	AttachedTags []*reps.Tag `json:"attached_tags"`

	AttachedTexts []*reps.Text `json:"attached_texts"`

	AttachedTimeIss []*reps.TimeIs `json:"attached_timeiss"`
}
