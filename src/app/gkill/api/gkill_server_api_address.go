package api

type GkillServerAPIAddress struct {
	LoginAddress string

	LogoutAddress string

	ResetPasswordAddress string

	SetNewPasswordAddress string

	AddTagAddress string

	AddTextAddress string

	AddNotificationAddress string

	AddKmemoAddress string

	AddURLogAddress string

	AddNlogAddress string

	AddTimeisAddress string

	AddMiAddress string

	AddLantanaAddress string

	AddRekyouAddress string

	UpdateTagAddress string

	UpdateTextAddress string

	UpdateNotificationAddress string

	UpdateKmemoAddress string

	UpdateURLogAddress string

	UpdateNlogAddress string

	UpdateTimeisAddress string

	UpdateLantanaAddress string

	UpdateIDFKyouAddress string

	UpdateMiAddress string

	UpdateRekyouAddress string

	GetKyousAddress string

	GetKyouAddress string

	GetKmemoAddress string

	GetURLogAddress string

	GetNlogAddress string

	GetTimeisAddress string

	GetMiAddress string

	GetLantanaAddress string

	GetRekyouAddress string

	GetGitCommitLogAddress string

	GetIDFKyouAddress string

	GetMiBoardListAddress string

	GetPlaingTimeisAddress string

	GetAllTagNamesAddress string

	GetAllRepNamesAddress string

	GetTagsByTargetIDAddress string

	GetTagHistoriesByTagIDAddress string

	GetTextsByTargetIDAddress string

	GetNotificationsByTargetIDAddress string

	GetTextHistoriesByTextIDAddress string

	GetNotificationHistoriesByNotificationIDAddress string

	GetApplicationConfigAddress string

	GetServerConfigsAddress string

	UploadFilesAddress string

	UploadGPSLogFilesAddress string

	UpdateApplicationConfigAddress string

	UpdateTagStructAddress string

	UpdateRepStructAddress string

	UpdateDeviceStructAddress string

	UpdateRepTypeStructAddress string

	UpdateKFTLTemplateAddress string

	UpdateAccountStatusAddress string

	UpdateUserRepsAddress string

	UpdateServerConfigsAddress string

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

	GetRepositoriesAddress string

	GetGkillNotificationPublicKeyAddress string

	RegisterGkillNotificationAddress string

	GkillWebpushServiceWorkerJsAddress string

	LoginMethod string

	LogoutMethod string

	ResetPasswordMethod string

	SetNewPasswordMethod string

	AddTagMethod string

	AddTextMethod string

	AddNotificationMethod string

	AddKmemoMethod string

	AddURLogMethod string

	AddNlogMethod string

	AddTimeisMethod string

	AddMiMethod string

	AddLantanaMethod string

	AddRekyouMethod string

	UpdateTagMethod string

	UpdateTextMethod string

	UpdateNotificationMethod string

	UpdateKmemoMethod string

	UpdateNlogMethod string

	UpdateURLogMethod string

	UpdateTimeisMethod string

	UpdateMiMethod string

	UpdateLantanaMethod string

	UpdateIDFKyouMethod string

	UpdateRekyouMethod string

	GetKyousMethod string

	GetKmemoMethod string

	GetURLogMethod string

	GetNlogMethod string

	GetTimeisMethod string

	GetMiMethod string

	GetLantanaMethod string

	GetRekyouMethod string

	GetKyouMethod string

	GetGitCommitLogMethod string

	GetIDFKyouMethod string

	GetMiBoardListMethod string

	GetPlaingTimeisMethod string

	GetAllTagNamesMethod string

	GetAllRepNamesMethod string

	GetTagsByTargetIDMethod string

	GetTagHistoriesByTagIDMethod string

	GetTextsByTargetIDMethod string

	GetNotificationsByTargetIDMethod string

	GetTextHistoriesByTagIDMethod string

	GetNotificationHistoriesByTagIDMethod string

	GetApplicationConfigMethod string

	GetServerConfigsMethod string

	UploadFilesMethod string

	UploadGPSLogFilesMethod string

	UpdateApplicationConfigMethod string

	UpdateTagStructMethod string

	UpdateRepStructMethod string

	UpdateDeviceStructMethod string

	UpdateRepTypeStructMethod string

	UpdateKFTLTemplateStructMethod string

	UpdateAccountStatusMethod string

	UpdateUserRepsMethod string

	UpdateServerConfigsMethod string

	AddAccountMethod string

	GenerateTLSFileMethod string

	GetGPSLogMethod string

	GetKFTLTemplateMethod string

	GetGkillInfoMethod string

	GetShareMiTaskListInfosMethod string

	AddShareMiTaskListInfoMethod string

	UpdateShareMiTaskListInfoMethod string

	DeleteShareMiTaskListInfosMethod string

	GetMiSharedTasksMethod string

	GetRepositoriesMethod string

	GetGkillNotificationPublicKeyMethod string

	RegisterGkillNotificationMethod string
}

func NewGKillAPIAddress() *GkillServerAPIAddress {

	gkillAPIAddress := &GkillServerAPIAddress{}
	gkillAPIAddress.LoginAddress = "/api/login"
	gkillAPIAddress.LogoutAddress = "/api/logout"
	gkillAPIAddress.ResetPasswordAddress = "/api/reset_password"
	gkillAPIAddress.SetNewPasswordAddress = "/api/set_new_password"
	gkillAPIAddress.AddTagAddress = "/api/add_tag"
	gkillAPIAddress.AddTextAddress = "/api/add_text"
	gkillAPIAddress.AddNotificationAddress = "/api/add_gkill_notification"
	gkillAPIAddress.AddKmemoAddress = "/api/add_kmemo"
	gkillAPIAddress.AddURLogAddress = "/api/add_urlog"
	gkillAPIAddress.AddNlogAddress = "/api/add_nlog"
	gkillAPIAddress.AddTimeisAddress = "/api/add_timeis"
	gkillAPIAddress.AddMiAddress = "/api/add_mi"
	gkillAPIAddress.AddLantanaAddress = "/api/add_lantana"
	gkillAPIAddress.AddRekyouAddress = "/api/add_rekyou"
	gkillAPIAddress.UpdateTagAddress = "/api/update_tag"
	gkillAPIAddress.UpdateTextAddress = "/api/update_text"
	gkillAPIAddress.UpdateNotificationAddress = "/api/update_gkill_notification"
	gkillAPIAddress.UpdateKmemoAddress = "/api/update_kmemo"
	gkillAPIAddress.UpdateURLogAddress = "/api/update_urlog"
	gkillAPIAddress.UpdateNlogAddress = "/api/update_nlog"
	gkillAPIAddress.UpdateTimeisAddress = "/api/update_timeis"
	gkillAPIAddress.UpdateLantanaAddress = "/api/update_lantana"
	gkillAPIAddress.UpdateIDFKyouAddress = "/api/update_idf_kyou"
	gkillAPIAddress.UpdateMiAddress = "/api/update_mi"
	gkillAPIAddress.UpdateRekyouAddress = "/api/update_rekyou"
	gkillAPIAddress.GetKyousAddress = "/api/get_kyous"
	gkillAPIAddress.GetKyouAddress = "/api/get_kyou"
	gkillAPIAddress.GetKmemoAddress = "/api/get_kmemo"
	gkillAPIAddress.GetURLogAddress = "/api/get_urlog"
	gkillAPIAddress.GetNlogAddress = "/api/get_nlog"
	gkillAPIAddress.GetTimeisAddress = "/api/get_timeis"
	gkillAPIAddress.GetMiAddress = "/api/get_mi"
	gkillAPIAddress.GetLantanaAddress = "/api/get_lantana"
	gkillAPIAddress.GetRekyouAddress = "/api/get_rekyou"
	gkillAPIAddress.GetGitCommitLogAddress = "/api/get_git_commit_log"
	gkillAPIAddress.GetIDFKyouAddress = "/api/get_idf_kyou"
	gkillAPIAddress.GetMiBoardListAddress = "/api/get_mi_board_list"
	gkillAPIAddress.GetPlaingTimeisAddress = "/api/get_plaing_timeis"
	gkillAPIAddress.GetAllTagNamesAddress = "/api/get_all_tag_names"
	gkillAPIAddress.GetAllRepNamesAddress = "/api/get_all_rep_names"
	gkillAPIAddress.GetTagsByTargetIDAddress = "/api/get_tags_by_id"
	gkillAPIAddress.GetTagHistoriesByTagIDAddress = "/api/get_tag_histories_by_tag_id"
	gkillAPIAddress.GetTextsByTargetIDAddress = "/api/get_texts_by_id"
	gkillAPIAddress.GetNotificationsByTargetIDAddress = "/api/get_gkill_notifications_by_id"
	gkillAPIAddress.GetTextHistoriesByTextIDAddress = "/api/get_text_histories_by_text_id"
	gkillAPIAddress.GetNotificationHistoriesByNotificationIDAddress = "/api/get_gkill_notification_histories_by_notification_id"
	gkillAPIAddress.GetApplicationConfigAddress = "/api/get_application_config"
	gkillAPIAddress.GetServerConfigsAddress = "/api/get_server_configs"
	gkillAPIAddress.UploadFilesAddress = "/api/upload_files"
	gkillAPIAddress.UploadGPSLogFilesAddress = "/api/upload_gpslog_files"
	gkillAPIAddress.UpdateApplicationConfigAddress = "/api/update_application_config"
	gkillAPIAddress.UpdateTagStructAddress = "/api/update_tag_struct"
	gkillAPIAddress.UpdateRepStructAddress = "/api/update_rep_struct"
	gkillAPIAddress.UpdateDeviceStructAddress = "/api/update_device_struct"
	gkillAPIAddress.UpdateRepTypeStructAddress = "/api/update_rep_type_struct"
	gkillAPIAddress.UpdateKFTLTemplateAddress = "/api/update_kftl_template"
	gkillAPIAddress.UpdateAccountStatusAddress = "/api/update_account_status"
	gkillAPIAddress.UpdateUserRepsAddress = "/api/update_user_reps"
	gkillAPIAddress.UpdateServerConfigsAddress = "/api/update_server_configs"
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
	gkillAPIAddress.GetRepositoriesAddress = "/api/get_repositories"
	gkillAPIAddress.GetGkillNotificationPublicKeyAddress = "/api/get_gkill_notification_public_key"
	gkillAPIAddress.RegisterGkillNotificationAddress = "/api/register_gkill_notification"
	gkillAPIAddress.GkillWebpushServiceWorkerJsAddress = "/serviceWorker.js"
	gkillAPIAddress.LoginMethod = "POST"
	gkillAPIAddress.LogoutMethod = "POST"
	gkillAPIAddress.ResetPasswordMethod = "POST"
	gkillAPIAddress.SetNewPasswordMethod = "POST"
	gkillAPIAddress.AddTagMethod = "POST"
	gkillAPIAddress.AddTextMethod = "POST"
	gkillAPIAddress.AddNotificationMethod = "POST"
	gkillAPIAddress.AddKmemoMethod = "POST"
	gkillAPIAddress.AddURLogMethod = "POST"
	gkillAPIAddress.AddNlogMethod = "POST"
	gkillAPIAddress.AddTimeisMethod = "POST"
	gkillAPIAddress.AddMiMethod = "POST"
	gkillAPIAddress.AddLantanaMethod = "POST"
	gkillAPIAddress.AddRekyouMethod = "POST"
	gkillAPIAddress.UpdateTagMethod = "POST"
	gkillAPIAddress.UpdateTextMethod = "POST"
	gkillAPIAddress.UpdateNotificationMethod = "POST"
	gkillAPIAddress.UpdateKmemoMethod = "POST"
	gkillAPIAddress.UpdateNlogMethod = "POST"
	gkillAPIAddress.UpdateURLogMethod = "POST"
	gkillAPIAddress.UpdateTimeisMethod = "POST"
	gkillAPIAddress.UpdateMiMethod = "POST"
	gkillAPIAddress.UpdateLantanaMethod = "POST"
	gkillAPIAddress.UpdateIDFKyouMethod = "POST"
	gkillAPIAddress.UpdateRekyouMethod = "POST"
	gkillAPIAddress.GetKyousMethod = "POST"
	gkillAPIAddress.GetKmemoMethod = "POST"
	gkillAPIAddress.GetURLogMethod = "POST"
	gkillAPIAddress.GetNlogMethod = "POST"
	gkillAPIAddress.GetTimeisMethod = "POST"
	gkillAPIAddress.GetMiMethod = "POST"
	gkillAPIAddress.GetLantanaMethod = "POST"
	gkillAPIAddress.GetKyouMethod = "POST"
	gkillAPIAddress.GetRekyouMethod = "POST"
	gkillAPIAddress.GetGitCommitLogMethod = "POST"
	gkillAPIAddress.GetIDFKyouMethod = "POST"
	gkillAPIAddress.GetMiBoardListMethod = "POST"
	gkillAPIAddress.GetPlaingTimeisMethod = "POST"
	gkillAPIAddress.GetAllTagNamesMethod = "POST"
	gkillAPIAddress.GetAllRepNamesMethod = "POST"
	gkillAPIAddress.GetTagsByTargetIDMethod = "POST"
	gkillAPIAddress.GetTagHistoriesByTagIDMethod = "POST"
	gkillAPIAddress.GetTextsByTargetIDMethod = "POST"
	gkillAPIAddress.GetNotificationsByTargetIDMethod = "POST"
	gkillAPIAddress.GetTextHistoriesByTagIDMethod = "POST"
	gkillAPIAddress.GetNotificationHistoriesByTagIDMethod = "POST"
	gkillAPIAddress.GetApplicationConfigMethod = "POST"
	gkillAPIAddress.GetServerConfigsMethod = "POST"
	gkillAPIAddress.UploadFilesMethod = "POST"
	gkillAPIAddress.UploadGPSLogFilesMethod = "POST"
	gkillAPIAddress.UpdateApplicationConfigMethod = "POST"
	gkillAPIAddress.UpdateTagStructMethod = "POST"
	gkillAPIAddress.UpdateRepStructMethod = "POST"
	gkillAPIAddress.UpdateDeviceStructMethod = "POST"
	gkillAPIAddress.UpdateRepTypeStructMethod = "POST"
	gkillAPIAddress.UpdateKFTLTemplateStructMethod = "POST"
	gkillAPIAddress.UpdateAccountStatusMethod = "POST"
	gkillAPIAddress.UpdateUserRepsMethod = "POST"
	gkillAPIAddress.UpdateServerConfigsMethod = "POST"
	gkillAPIAddress.AddAccountMethod = "POST"
	gkillAPIAddress.GenerateTLSFileMethod = "POST"
	gkillAPIAddress.GetGPSLogMethod = "POST"
	gkillAPIAddress.GetKFTLTemplateMethod = "POST"
	gkillAPIAddress.GetGkillInfoMethod = "POST"
	gkillAPIAddress.GetShareMiTaskListInfosMethod = "POST"
	gkillAPIAddress.AddShareMiTaskListInfoMethod = "POST"
	gkillAPIAddress.UpdateShareMiTaskListInfoMethod = "POST"
	gkillAPIAddress.DeleteShareMiTaskListInfosMethod = "POST"
	gkillAPIAddress.GetMiSharedTasksMethod = "POST"
	gkillAPIAddress.GetRepositoriesMethod = "POST"
	gkillAPIAddress.GetGkillNotificationPublicKeyMethod = "POST"
	gkillAPIAddress.RegisterGkillNotificationMethod = "POST"
	return gkillAPIAddress
}
