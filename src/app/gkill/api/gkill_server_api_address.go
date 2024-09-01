// ˅
package api

// ˄

type GkillServerAPIAddress struct {
	// ˅

	// ˄

	LoginAddress string

	LogoutAddress string

	ResetPasswordAddress string

	SetNewPasswordAddress string

	AddTagAddress string

	AddTextAddress string

	AddKmemoAddress string

	AddURLogAddress string

	AddNlogAddress string

	AddTimeisAddress string

	AddMiAddress string

	AddLantanaAddress string

	AddKyouInfoAddress string

	AddRekyouAddress string

	UpdateTagAddress string

	UpdateTextAddress string

	UpdateKmemoAddress string

	UpdateURLogAddress string

	UpdateNlogAddress string

	UpdateTimeisAddress string

	UpdateLantanaAddress string

	UpdateMiAddress string

	UpdateKyouInfoAddress string

	UpdateRekyouAddress string

	GetKyousAddress string

	GetKyouAddress string

	GetKmemosAddress string

	GetURLogsAddress string

	GetNlogsAddress string

	GetTimeissAddress string

	GetRekyousAddress string

	GetMisAddress string

	GetLantanasAddress string

	GetKmemoAddress string

	GetURLogAddress string

	GetNlogAddress string

	GetTimeisAddress string

	GetMiAddress string

	GetLantanaAddress string

	GetRekyouAddress string

	GetGitCommitLogAddress string

	GetGitCommitLogsAddress string

	GetMiBoardListAddress string

	GetPlaingTimeisAddress string

	GetAllTagNamesAddress string

	GetTagsByTargetIDAddress string

	GetTagHistoriesByTagIDAddress string

	GetTextsByTargetIDAddress string

	GetTextHistoriesByTextIDAddress string

	GetApplicationConfigAddress string

	GetServerConfigAddress string

	UploadFilesAddress string

	UploadGPSLogFilesAddress string

	UpdateTagStructAddress string

	UpdateRepStructAddress string

	UpdateDeviceStructAddress string

	UpdateRepTypeStructAddress string

	UpdateAccountStatusAddress string

	UpdateUserRepsAddress string

	AddAccountAddress string

	GenerateTLSFileAddress string

	GetGPSLogAddress string

	GetKFTLTemplateAddress string

	GetGkillInfoAddress string

	GetShareMiTaskListInfosAddress string

	AddShareMiTaskListInfoAddress string

	UpdateShareMiTaskListInfoAddress string

	DeleteShareMiTaskListInfosAddress string

	GetMiSharedTasksAddress string

	LoginMethod string

	LogoutMethod string

	SetNewPasswordMethod string

	AddTagMethod string

	AddTextMethod string

	AddKmemoMethod string

	AddURLogMethod string

	AddNlogMethod string

	AddTimeisMethod string

	AddMiMethod string

	AddLantanaMethod string

	AddKyouInfoMethod string

	AddRekyouMethod string

	UpdateTagMethod string

	UpdateTextMethod string

	UpdateKmemoMethod string

	UpdateNlogMethod string

	UpdateURLogMethod string

	UpdateTimeisMethod string

	UpdateMiMethod string

	UpdateLantanaMethod string

	UpdateKyouInfoMethod string

	UpdateRekyouMethod string

	GetKyousMethod string

	GetKyouMethod string

	GetRekyouMethod string

	GetRekyousMethod string

	GetGitCommitLogsMethod string

	GetMiNoardListMethod string

	GetPlaingTimeisMethod string

	GetAllTagNamesMethod string

	GetTagsByTargetIDMethod string

	GetTagHistoriesByTagIDMethod string

	GetTextsByTargetIDMethod string

	GetTextHistoriesByTagIDMethod string

	GetApplicationConfigMethod string

	GetServerConfigMethod string

	UploadFilesMethod string

	UploadGPSLogFilesMethod string

	UpdateTagStructMethod string

	UpdateRepStructMethod string

	UpdateDeviceStructMethod string

	UpdateRepTypeStructMethod string

	UpdateAccountStatusMethod string

	UpdateUserRepsMethod string

	AddAccountMethod string

	GenerateTLSFileMethod string

	GetGPSLogMethod string

	GetKFTLTemplateMethod string

	GetGkillInfoMethod string

	GetShareMiTaskListInfosMethod string

	AddShareMiTaskListInfoMethod string

	UpdateShareMiTaskListInfoMethod string

	DeleteShareMiTaskListInfosMethod string

	// ˅

	// ˄
}

func NewGKillAPIAddress() *GkillServerAPIAddress {
	// ˅

	gkillAPIAddress := &GkillServerAPIAddress{}
	gkillAPIAddress.LoginAddress = "/api/login"
	gkillAPIAddress.LogoutAddress = "/api/logout"
	gkillAPIAddress.ResetPasswordAddress = "/api/reset_password"
	gkillAPIAddress.SetNewPasswordAddress = "/api/set_new_password"
	gkillAPIAddress.AddTagAddress = "/api/add_tag"
	gkillAPIAddress.AddTextAddress = "/api/add_text"
	gkillAPIAddress.AddKmemoAddress = "/api/add_kmemo"
	gkillAPIAddress.AddURLogAddress = "/api/add_urlog"
	gkillAPIAddress.AddNlogAddress = "/api/add_nlog"
	gkillAPIAddress.AddTimeisAddress = "/api/add_timeis"
	gkillAPIAddress.AddMiAddress = "/api/add_mi"
	gkillAPIAddress.AddLantanaAddress = "/api/add_lantana"
	gkillAPIAddress.AddKyouInfoAddress = "/api/add_kyou_info"
	gkillAPIAddress.AddRekyouAddress = "/api/add_rekyou"
	gkillAPIAddress.UpdateTagAddress = "/api/update_tag"
	gkillAPIAddress.UpdateTextAddress = "/api/update_text"
	gkillAPIAddress.UpdateKmemoAddress = "/api/update_kmemo"
	gkillAPIAddress.UpdateURLogAddress = "/api/update_urlog"
	gkillAPIAddress.UpdateNlogAddress = "/api/update_nlog"
	gkillAPIAddress.UpdateTimeisAddress = "/api/update_timeis"
	gkillAPIAddress.UpdateLantanaAddress = "/api/update_lantana"
	gkillAPIAddress.UpdateMiAddress = "/api/update_mi"
	gkillAPIAddress.UpdateKyouInfoAddress = "/api/update_kyou_info"
	gkillAPIAddress.UpdateRekyouAddress = "/api/update_rekyou"
	gkillAPIAddress.GetKyousAddress = "/api/get_kyous"
	gkillAPIAddress.GetKyouAddress = "/api/get_kyou"
	gkillAPIAddress.GetKmemosAddress = "/api/get_kmemos"
	gkillAPIAddress.GetURLogsAddress = "/api/get_urlogs"
	gkillAPIAddress.GetNlogsAddress = "/api/get_nlogs"
	gkillAPIAddress.GetTimeissAddress = "/api/get_timeiss"
	gkillAPIAddress.GetRekyousAddress = "/api/get_rekyous"
	gkillAPIAddress.GetMisAddress = "/api/get_mis"
	gkillAPIAddress.GetLantanasAddress = "/api/get_lantanas"
	gkillAPIAddress.GetKmemoAddress = "/api/get_kmemo"
	gkillAPIAddress.GetURLogAddress = "/api/get_urlog"
	gkillAPIAddress.GetNlogAddress = "/api/get_nlog"
	gkillAPIAddress.GetTimeisAddress = "/api/get_timeis"
	gkillAPIAddress.GetMiAddress = "/api/get_mi"
	gkillAPIAddress.GetLantanaAddress = "/api/get_lantana"
	gkillAPIAddress.GetRekyouAddress = "/api/get_rekyou"
	gkillAPIAddress.GetGitCommitLogAddress = "/api/get_git_commit_log"
	gkillAPIAddress.GetGitCommitLogsAddress = "/api/get_git_commit_logs"
	gkillAPIAddress.GetMiBoardListAddress = "/api/get_mi_board_list"
	gkillAPIAddress.GetPlaingTimeisAddress = "/api/get_plaing_timeis"
	gkillAPIAddress.GetAllTagNamesAddress = "/api/get_all_tag_names"
	gkillAPIAddress.GetTagsByTargetIDAddress = "/api/get_tags_by_id"
	gkillAPIAddress.GetTagHistoriesByTagIDAddress = "/api/get_tag_histories_by_tag_id"
	gkillAPIAddress.GetTextsByTargetIDAddress = "/api/get_texts_by_id"
	gkillAPIAddress.GetTextHistoriesByTextIDAddress = "/api/get_text_histories_by_text_id"
	gkillAPIAddress.GetApplicationConfigAddress = "/api/get_application_config"
	gkillAPIAddress.GetServerConfigAddress = "/api/get_server_config"
	gkillAPIAddress.UploadFilesAddress = "/api/upload_files"
	gkillAPIAddress.UploadGPSLogFilesAddress = "/api/upload_gpslog_files"
	gkillAPIAddress.UpdateTagStructAddress = "/api/update_tag_struct"
	gkillAPIAddress.UpdateRepStructAddress = "/api/update_rep_struct"
	gkillAPIAddress.UpdateDeviceStructAddress = "/api/update_device_struct"
	gkillAPIAddress.UpdateRepTypeStructAddress = "/api/update_rep_struct"
	gkillAPIAddress.UpdateAccountStatusAddress = "/api/update_account_status"
	gkillAPIAddress.UpdateUserRepsAddress = "/api/update_user_reps"
	gkillAPIAddress.AddAccountAddress = "/api/add_user"
	gkillAPIAddress.GenerateTLSFileAddress = "/api/generate_tls_file"
	gkillAPIAddress.GetGPSLogAddress = "/api/get_gps_log"
	gkillAPIAddress.GetKFTLTemplateAddress = "/api/get_kftl_template"
	gkillAPIAddress.GetGkillInfoAddress = "/api/get_gkill_info"
	gkillAPIAddress.GetShareMiTaskListInfosAddress = "/api/get_share_mi_task_list_infos"
	gkillAPIAddress.AddShareMiTaskListInfoAddress = "/api/add_share_mi_task_list_info"
	gkillAPIAddress.UpdateShareMiTaskListInfoAddress = "/api/update_share_mi_task_list_info"
	gkillAPIAddress.DeleteShareMiTaskListInfosAddress = "/api/delete_share_mi_task_list_infos"
	gkillAPIAddress.GetMiSharedTasksAddress = "/api/get_mi_shared_tasks"
	gkillAPIAddress.LoginMethod = "POST"
	gkillAPIAddress.LogoutMethod = "POST"
	gkillAPIAddress.SetNewPasswordMethod = "POST"
	gkillAPIAddress.AddTagMethod = "POST"
	gkillAPIAddress.AddTextMethod = "POST"
	gkillAPIAddress.AddKmemoMethod = "POST"
	gkillAPIAddress.AddURLogMethod = "POST"
	gkillAPIAddress.AddNlogMethod = "POST"
	gkillAPIAddress.AddTimeisMethod = "POST"
	gkillAPIAddress.AddMiMethod = "POST"
	gkillAPIAddress.AddLantanaMethod = "POST"
	gkillAPIAddress.AddKyouInfoMethod = "POST"
	gkillAPIAddress.AddRekyouMethod = "POST"
	gkillAPIAddress.UpdateTagMethod = "POST"
	gkillAPIAddress.UpdateTextMethod = "POST"
	gkillAPIAddress.UpdateKmemoMethod = "POST"
	gkillAPIAddress.UpdateNlogMethod = "POST"
	gkillAPIAddress.UpdateURLogMethod = "POST"
	gkillAPIAddress.UpdateTimeisMethod = "POST"
	gkillAPIAddress.UpdateMiMethod = "POST"
	gkillAPIAddress.UpdateLantanaMethod = "POST"
	gkillAPIAddress.UpdateKyouInfoMethod = "POST"
	gkillAPIAddress.UpdateRekyouMethod = "POST"
	gkillAPIAddress.GetKyousMethod = "POST"
	gkillAPIAddress.GetKyouMethod = "POST"
	gkillAPIAddress.GetRekyouMethod = "POST"
	gkillAPIAddress.GetRekyousMethod = "POST"
	gkillAPIAddress.GetGitCommitLogsMethod = "POST"
	gkillAPIAddress.GetMiNoardListMethod = "POST"
	gkillAPIAddress.GetPlaingTimeisMethod = "POST"
	gkillAPIAddress.GetAllTagNamesMethod = "POST"
	gkillAPIAddress.GetTagsByTargetIDMethod = "POST"
	gkillAPIAddress.GetTagHistoriesByTagIDMethod = "POST"
	gkillAPIAddress.GetTextsByTargetIDMethod = "POST"
	gkillAPIAddress.GetTextHistoriesByTagIDMethod = "POST"
	gkillAPIAddress.GetApplicationConfigMethod = "POST"
	gkillAPIAddress.GetServerConfigMethod = "POST"
	gkillAPIAddress.UploadFilesMethod = "POST"
	gkillAPIAddress.UploadGPSLogFilesMethod = "POST"
	gkillAPIAddress.UpdateTagStructMethod = "POST"
	gkillAPIAddress.UpdateRepStructMethod = "POST"
	gkillAPIAddress.UpdateDeviceStructMethod = "POST"
	gkillAPIAddress.UpdateRepTypeStructMethod = "POST"
	gkillAPIAddress.UpdateAccountStatusMethod = "POST"
	gkillAPIAddress.UpdateUserRepsMethod = "POST"
	gkillAPIAddress.AddAccountMethod = "POST"
	gkillAPIAddress.GenerateTLSFileMethod = "POST"
	gkillAPIAddress.GetGPSLogMethod = "POST"
	gkillAPIAddress.GetKFTLTemplateMethod = "POST"
	gkillAPIAddress.GetGkillInfoMethod = "POST"
	gkillAPIAddress.GetShareMiTaskListInfosMethod = "POST"
	gkillAPIAddress.AddShareMiTaskListInfoMethod = "POST"
	gkillAPIAddress.UpdateShareMiTaskListInfoMethod = "POST"
	gkillAPIAddress.DeleteShareMiTaskListInfosMethod = "POST"

	return gkillAPIAddress
	// ˄
}

// ˅

// ˄
