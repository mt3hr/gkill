package req_res

type AddShareMiTaskListInfoRequest struct {
	SessionID string `json:"session_id"`

	ShareMiTaskListInfo *ShareMiTaskListInfo `json:"share_mi_task_list_info"`
}
