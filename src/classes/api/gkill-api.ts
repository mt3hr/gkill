'use strict'

import { ApplicationConfig } from "../datas/config/application-config"
import type { AddAccountRequest } from "./req_res/add-account-request"
import type { AddAccountResponse } from "./req_res/add-account-response"
import type { AddKmemoRequest } from "./req_res/add-kmemo-request"
import type { AddKmemoResponse } from "./req_res/add-kmemo-response"
import type { AddLantanaRequest } from "./req_res/add-lantana-request"
import type { AddLantanaResponse } from "./req_res/add-lantana-response"
import type { AddMiRequest } from "./req_res/add-mi-request"
import type { AddMiResponse } from "./req_res/add-mi-response"
import type { AddNlogRequest } from "./req_res/add-nlog-request"
import type { AddNlogResponse } from "./req_res/add-nlog-response"
import type { AddReKyouRequest } from "./req_res/add-re-kyou-request"
import type { AddReKyouResponse } from "./req_res/add-re-kyou-response"
import type { AddShareMiTaskListInfoRequest } from "./req_res/add-share-mi-task-list-info-request"
import { AddShareMiTaskListInfoResponse } from "./req_res/add-share-mi-task-list-info-response"
import type { AddTagRequest } from "./req_res/add-tag-request"
import type { AddTagResponse } from "./req_res/add-tag-response"
import type { AddTextRequest } from "./req_res/add-text-request"
import type { AddTextResponse } from "./req_res/add-text-response"
import type { AddTimeisRequest } from "./req_res/add-timeis-request"
import type { AddTimeisResponse } from "./req_res/add-timeis-response"
import type { AddURLogRequest } from "./req_res/add-ur-log-request"
import type { AddURLogResponse } from "./req_res/add-ur-log-response"
import type { DeleteShareMiTaskListInfosRequest } from "./req_res/delete-share-mi-task-list-infos-request"
import { DeleteShareMiTaskListInfosResponse } from "./req_res/delete-share-mi-task-list-infos-response"
import type { GenerateTLSFileRequest } from "./req_res/generate-tls-file-request"
import type { GenerateTLSFileResponse } from "./req_res/generate-tls-file-response"
import type { GetAllTagNamesRequest } from "./req_res/get-all-tag-names-request"
import type { GetAllTagNamesResponse } from "./req_res/get-all-tag-names-response"
import type { GetApplicationConfigRequest } from "./req_res/get-application-config-request"
import { GetApplicationConfigResponse } from "./req_res/get-application-config-response"
import type { GetGitCommitLogRequest } from "./req_res/get-git-commit-log-request"
import type { GetGitCommitLogResponse } from "./req_res/get-git-commit-log-response"
import { GetGkillInfoRequest } from "./req_res/get-gkill-info-request"
import { GetGkillInfoResponse } from "./req_res/get-gkill-info-response"
import type { GetGPSLogRequest } from "./req_res/get-gps-log-request"
import type { GetGPSLogResponse } from "./req_res/get-gps-log-response"
import type { GetKFTLTemplatesRequest } from "./req_res/get-kftl-templates-request"
import type { GetKFTLTemplatesResponse } from "./req_res/get-kftl-templates-response"
import type { GetKmemoRequest } from "./req_res/get-kmemo-request"
import type { GetKmemoResponse } from "./req_res/get-kmemo-response"
import type { GetKyouRequest } from "./req_res/get-kyou-request"
import { GetKyouResponse } from "./req_res/get-kyou-response"
import type { GetKyousRequest } from "./req_res/get-kyous-request"
import type { GetKyousResponse } from "./req_res/get-kyous-response"
import type { GetLantanaRequest } from "./req_res/get-lantana-request"
import type { GetLantanaResponse } from "./req_res/get-lantana-response"
import type { GetMiBoardRequest } from "./req_res/get-mi-board-request"
import type { GetMiBoardResponse } from "./req_res/get-mi-board-response"
import type { GetMiRequest } from "./req_res/get-mi-request"
import { GetMiResponse } from "./req_res/get-mi-response"
import type { GetNlogRequest } from "./req_res/get-nlog-request"
import type { GetNlogResponse } from "./req_res/get-nlog-response"
import type { GetPlaingTimeisRequest } from "./req_res/get-plaing-timeis-request"
import { GetPlaingTimeisResponse } from "./req_res/get-plaing-timeis-response"
import type { GetReKyouRequest } from "./req_res/get-re-kyou-request"
import type { GetReKyouResponse } from "./req_res/get-re-kyou-response"
import type { GetShareMiTaskListInfosRequest } from "./req_res/get-share-mi-task-list-infos-request"
import { GetShareMiTaskListInfosResponse } from "./req_res/get-share-mi-task-list-infos-response"
import type { GetSharedMiTasksRequest } from "./req_res/get-shared-mi-tasks-request"
import { GetSharedMiTasksResponse } from "./req_res/get-shared-mi-tasks-response"
import type { GetTagHistoryByTagIDRequest } from "./req_res/get-tag-history-by-tag-id-request"
import { GetTagHistoryByTagIDResponse } from "./req_res/get-tag-history-by-tag-id-response"
import type { GetTagsByTargetIDRequest } from "./req_res/get-tags-by-target-id-request"
import { GetTagsByTargetIDResponse } from "./req_res/get-tags-by-target-id-response"
import type { GetTextHistoryByTextIDRequest } from "./req_res/get-text-history-by-tag-id-request"
import { GetTextHistoryByTextIDResponse } from "./req_res/get-text-history-by-tag-id-response"
import type { GetTextsByTargetIDRequest } from "./req_res/get-texts-by-target-id-request"
import { GetTextsByTargetIDResponse } from "./req_res/get-texts-by-target-id-response"
import type { GetTimeisRequest } from "./req_res/get-timeis-request"
import { GetTimeisResponse } from "./req_res/get-timeis-response"
import type { GetURLogRequest } from "./req_res/get-ur-log-request"
import type { GetURLogResponse } from "./req_res/get-ur-log-response"
import type { LoginRequest } from "./req_res/login-request"
import type { LoginResponse } from "./req_res/login-response"
import type { LogoutRequest } from "./req_res/logout-request"
import type { LogoutResponse } from "./req_res/logout-response"
import type { ResetPasswordRequest } from "./req_res/reset-password-request"
import type { ResetPasswordResponse } from "./req_res/reset-password-response"
import type { SetNewPasswordRequest } from "./req_res/set-new-password-request"
import type { SetNewPasswordResponse } from "./req_res/set-new-password-response"
import type { UpdateAccountStatusRequest } from "./req_res/update-account-status-request"
import type { UpdateAccountStatusResponse } from "./req_res/update-account-status-response"
import type { UpdateApplicationConfigRequest } from "./req_res/update-application-config-request"
import type { UpdateApplicationConfigResponse } from "./req_res/update-application-config-response"
import type { UpdateDeviceStructRequest } from "./req_res/update-device-struct-request"
import type { UpdateDeviceStructResponse } from "./req_res/update-device-struct-response"
import type { UpdateKmemoRequest } from "./req_res/update-kmemo-request"
import type { UpdateKmemoResponse } from "./req_res/update-kmemo-response"
import type { UpdateLantanaRequest } from "./req_res/update-lantana-request"
import type { UpdateLantanaResponse } from "./req_res/update-lantana-response"
import type { UpdateMiRequest } from "./req_res/update-mi-request"
import type { UpdateMiResponse } from "./req_res/update-mi-response"
import type { UpdateNlogRequest } from "./req_res/update-nlog-request"
import type { UpdateNlogResponse } from "./req_res/update-nlog-response"
import type { UpdateReKyouRequest } from "./req_res/update-re-kyou-request"
import type { UpdateReKyouResponse } from "./req_res/update-re-kyou-response"
import type { UpdateRepStructRequest } from "./req_res/update-rep-struct-request"
import type { UpdateRepStructResponse } from "./req_res/update-rep-struct-response"
import type { UpdateRepTypeStructRequest } from "./req_res/update-rep-type-struct-request"
import type { UpdateRepTypeStructResponse } from "./req_res/update-rep-type-struct-response"
import type { UpdateKFTLTemplateRequest } from "./req_res/update-kftl-template-request.ts"
import type { UpdateKFTLTemplateResponse } from "./req_res/update-kftl-template-response.ts"
import type { UpdateShareMiTaskListInfoRequest } from "./req_res/update-share-mi-task-list-info-request"
import { UpdateShareMiTaskListInfoResponse } from "./req_res/update-share-mi-task-list-info-response"
import type { UpdateTagRequest } from "./req_res/update-tag-request"
import type { UpdateTagResponse } from "./req_res/update-tag-response"
import type { UpdateTagStructRequest } from "./req_res/update-tag-struct-request"
import type { UpdateTagStructResponse } from "./req_res/update-tag-struct-response"
import type { UpdateTextRequest } from "./req_res/update-text-request"
import type { UpdateTextResponse } from "./req_res/update-text-response"
import type { UpdateTimeisRequest } from "./req_res/update-timeis-request"
import type { UpdateTimeisResponse } from "./req_res/update-timeis-response"
import type { UpdateURLogRequest } from "./req_res/update-ur-log-request"
import type { UpdateURLogResponse } from "./req_res/update-ur-log-response"
import type { UpdateUserRepsRequest } from "./req_res/update-user-reps-request"
import type { UpdateUserRepsResponse } from "./req_res/update-user-reps-response"
import type { UploadFilesRequest } from "./req_res/upload-files-request"
import type { UploadFilesResponse } from "./req_res/upload-files-response"
import type { UploadGPSLogFilesRequest } from "./req_res/upload-gps-log-files-request"
import type { UploadGPSLogFilesResponse } from "./req_res/upload-gps-log-files-response"
import type { GkillAPIResponse } from "./gkill-api-response"
import router from "@/router"
import type { GetRepositoriesResponse } from "./req_res/get-repositories-response"
import type { GetRepositoriesRequest } from "./req_res/get-repositories-request"
import type { GetAllRepNamesRequest } from "./req_res/get-all-rep-names-request"
import type { GetAllRepNamesResponse } from "./req_res/get-all-rep-names-response"
import { Kyou } from "../datas/kyou"
import moment from "moment"
import { Tag } from "../datas/tag"
import { Text } from "../datas/text"
import { Notification } from "../datas/notification"
import { Kmemo } from "../datas/kmemo"
import { URLog } from "../datas/ur-log"
import { Nlog } from "../datas/nlog"
import { TimeIs } from "../datas/time-is"
import { Mi } from "../datas/mi"
import { Lantana } from "../datas/lantana"
import { ReKyou } from "../datas/re-kyou"
import { GitCommitLog } from "../datas/git-commit-log"
import { IDFKyou } from "../datas/idf-kyou"
import type { GetIDFKyouRequest } from "./req_res/get-idf-kyou-request"
import type { GetIDFKyouResponse } from "./req_res/get-idf-kyou-response"
import { GPSLog } from "../datas/gps-log"
import { FindKyouQuery } from "./find_query/find-kyou-query"
import type { UpdateIDFKyouRequest } from "./req_res/update-idf-kyou-request"
import type { UpdateIDFKyouResponse } from "./req_res/update-idf-kyou-response"
import type { GetServerConfigsRequest } from "./req_res/get-server-configs-request"
import type { GetServerConfigsResponse } from "./req_res/get-server-configs-response"
import type { UpdateServerConfigsRequest } from "./req_res/update-server-configs-request"
import type { UpdateServerConfigsResponse } from "./req_res/update-server-configs-response"
import { ShareMiTaskListInfo } from "../datas/share-mi-task-list-info"
import type { GetGkillNotificationPublicKeyRequest } from "./req_res/get-gkill-notification-public-key-request"
import type { GetGkillNotificationPublicKeyResponse } from "./req_res/get-gkill-notification-public-key-response"
import type { RegisterGkillNotificationRequest } from "./req_res/register-gkill-notification-request"
import type { RegisterGkillNotificationResponse } from "./req_res/register-gkill-notification-response"
import type { AddNotificationRequest } from "./req_res/add-notification-request"
import type { AddNotificationResponse } from "./req_res/add-notification-response"
import type { GetNotificationHistoryByNotificationIDRequest } from "./req_res/get-notification-history-by-notification-id-request copy"
import type { GetNotificationHistoryByNotificationIDResponse } from "./req_res/get-notification-history-by-notification-id-response"
import type { GetNotificationsByTargetIDRequest } from "./req_res/get-notifications-by-target-id-request"
import type { GetNotificationsByTargetIDResponse } from "./req_res/get-notifications-by-target-id-response"
import type { UpdateNotificationRequest } from "./req_res/update-notification-request"
import type { UpdateNotificationResponse } from "./req_res/update-notification-response"
import type { OpenFileRequest } from "./req_res/open-file-request"
import type { OpenFileResponse } from "./req_res/open-file-response"
import type { OpenDirectoryRequest } from "./req_res/open-directory-request"
import type { OpenDirectoryResponse } from "./req_res/open-directory-response"
import type { ReloadRepositoriesRequest } from "./req_res/reload-repositories-request"
import type { ReloadRepositoriesResponse } from "./req_res/reload-repositories-response"

export class GkillAPI {
        // 画面以外から参照されるやつ
        private static use_gkill_api: GkillAPI | null
        static get_gkill_api(): GkillAPI {
                const api = this.use_gkill_api
                if (!api) {
                        return this.get_instance()
                }
                return api
        }
        static set_gkill_api(gkill_api: GkillAPI): void {
                GkillAPI.use_gkill_api = gkill_api
        }

        private static gkill_api: GkillAPI = new GkillAPI()
        static get_instance(): GkillAPI {
                return this.gkill_api
        }

        login_address: string
        logout_address: string
        reset_password_address: string
        set_new_password_address: string
        add_tag_address: string
        add_text_address: string
        add_notification_address: string
        add_kmemo_address: string
        add_urlog_address: string
        add_nlog_address: string
        add_timeis_address: string
        add_mi_address: string
        add_lantana_address: string
        add_rekyou_address: string
        update_tag_address: string
        update_text_address: string
        update_notification_address: string
        update_kmemo_address: string
        update_urlog_address: string
        update_nlog_address: string
        update_timeis_address: string
        update_lantana_address: string
        update_idf_kyou_address: string
        update_mi_address: string
        update_rekyou_address: string
        get_kyous_address: string
        get_kyou_address: string
        get_kmemo_address: string
        get_urlog_address: string
        get_nlog_address: string
        get_timeis_address: string
        get_mi_address: string
        get_lantana_address: string
        get_rekyou_address: string
        get_git_commit_log_address: string
        get_idf_kyou_address: string
        get_mi_board_list_address: string
        get_plaing_timeis_address: string
        get_all_tag_names_address: string
        get_all_rep_names_address: string
        get_tags_by_target_id_address: string
        get_tag_histories_by_tag_id_address: string
        get_texts_by_target_id_address: string
        get_notifications_by_target_id_address: string
        get_text_histories_by_text_id_address: string
        get_notification_histories_by_notification_id_address: string
        get_application_config_address: string
        get_server_configs_address: string
        update_application_config_address: string
        upload_files_address: string
        upload_gpslog_files_address: string
        update_tag_struct_address: string
        update_rep_struct_address: string
        update_device_struct_address: string
        update_rep_type_struct_address: string
        update_kftl_template_address: string
        update_account_status_address: string
        update_user_reps_address: string
        update_server_configs_address: string
        add_account_address: string
        generate_tls_file_address: string
        get_gps_log_address: string
        get_kftl_template_address: string
        get_gkill_info_address: string
        get_share_mi_task_list_infos_address: string
        add_share_mi_task_list_info_address: string
        update_share_mi_task_list_info_address: string
        delete_share_mi_task_list_infos_address: string
        get_mi_shared_tasks_address: string
        get_repositories_address: string
        get_gkill_notification_public_key_address: string
        register_gkill_notification_address: string
        gkill_webpush_service_worker_js_address: string
        open_directory_address: string
        open_file_address: string
        reload_repositories_address: string

        login_method: string
        logout_method: string
        reset_password_method: string
        set_new_password_method: string
        add_tag_method: string
        add_text_method: string
        add_notification_method: string
        add_kmemo_method: string
        add_urlog_method: string
        add_nlog_method: string
        add_timeis_method: string
        add_mi_method: string
        add_lantana_method: string
        add_rekyou_method: string
        update_tag_method: string
        update_text_method: string
        update_notification_method: string
        update_kmemo_method: string
        update_nlog_method: string
        update_urlog_method: string
        update_timeis_method: string
        update_mi_method: string
        update_lantana_method: string
        update_idf_kyou_method: string
        update_rekyou_method: string
        get_kyous_method: string
        get_kyou_method: string
        get_kmemo_method: string
        get_urlog_method: string
        get_nlog_method: string
        get_timeis_method: string
        get_mi_method: string
        get_lantana_method: string
        get_git_commit_log_method: string
        get_idf_kyou_method: string
        get_rekyou_method: string
        get_rekyous_method: string
        get_git_commit_logs_method: string
        get_mi_board_list_method: string
        get_plaing_timeis_method: string
        get_all_tag_names_method: string
        get_all_rep_names_method: string
        get_tags_by_target_id_method: string
        get_tag_histories_by_tag_id_method: string
        get_texts_by_target_id_method: string
        get_notifications_by_target_id_method: string
        get_text_histories_by_text_id_method: string
        get_notification_histories_by_notification_id_method: string
        get_application_config_method: string
        get_server_configs_method: string
        update_application_config_method: string
        upload_files_method: string
        upload_gpslog_files_method: string
        update_tag_struct_method: string
        update_rep_struct_method: string
        update_device_struct_method: string
        update_rep_type_struct_method: string
        update_kftl_template_method: string
        update_account_status_method: string
        update_user_reps_method: string
        update_server_config_method: string
        add_account_method: string
        generate_tls_file_method: string
        get_gps_log_method: string
        get_kftl_template_method: string
        get_gkill_info_method: string
        get_share_mi_task_list_infos_method: string
        add_share_mi_task_list_info_method: string
        update_share_mi_task_list_info_method: string
        delete_share_mi_task_list_infos_method: string
        get_mi_shared_tasks_method: string
        get_repositories_method: string
        get_gkill_notification_public_key_method: string
        register_gkill_notification_method: string
        open_directory_method: string
        open_file_method: string
        reload_repositories_method: string

        protected constructor() {
                this.saved_application_config = null
                this.login_address = "/api/login"
                this.logout_address = "/api/logout"
                this.reset_password_address = "/api/reset_password"
                this.set_new_password_address = "/api/set_new_password"
                this.add_tag_address = "/api/add_tag"
                this.add_text_address = "/api/add_text"
                this.add_notification_address = "/api/add_gkill_notification"
                this.add_kmemo_address = "/api/add_kmemo"
                this.add_urlog_address = "/api/add_urlog"
                this.add_nlog_address = "/api/add_nlog"
                this.add_timeis_address = "/api/add_timeis"
                this.add_mi_address = "/api/add_mi"
                this.add_lantana_address = "/api/add_lantana"
                this.add_rekyou_address = "/api/add_rekyou"
                this.update_tag_address = "/api/update_tag"
                this.update_text_address = "/api/update_text"
                this.update_notification_address = "/api/update_gkill_notification"
                this.update_kmemo_address = "/api/update_kmemo"
                this.update_urlog_address = "/api/update_urlog"
                this.update_nlog_address = "/api/update_nlog"
                this.update_timeis_address = "/api/update_timeis"
                this.update_lantana_address = "/api/update_lantana"
                this.update_idf_kyou_address = "/api/update_idf_kyou"
                this.update_mi_address = "/api/update_mi"
                this.update_rekyou_address = "/api/update_rekyou"
                this.get_kyous_address = "/api/get_kyous"
                this.get_kyou_address = "/api/get_kyou"
                this.get_kmemo_address = "/api/get_kmemo"
                this.get_urlog_address = "/api/get_urlog"
                this.get_nlog_address = "/api/get_nlog"
                this.get_timeis_address = "/api/get_timeis"
                this.get_mi_address = "/api/get_mi"
                this.get_lantana_address = "/api/get_lantana"
                this.get_rekyou_address = "/api/get_rekyou"
                this.get_git_commit_log_address = "/api/get_git_commit_log"
                this.get_idf_kyou_address = "/api/get_idf_kyou"
                this.get_mi_board_list_address = "/api/get_mi_board_list"
                this.get_plaing_timeis_address = "/api/get_plaing_timeis"
                this.get_all_tag_names_address = "/api/get_all_tag_names"
                this.get_all_rep_names_address = "/api/get_all_rep_names"
                this.get_tags_by_target_id_address = "/api/get_tags_by_id"
                this.get_tag_histories_by_tag_id_address = "/api/get_tag_histories_by_tag_id"
                this.get_texts_by_target_id_address = "/api/get_texts_by_id"
                this.get_notifications_by_target_id_address = "/api/get_gkill_notifications_by_id"
                this.get_text_histories_by_text_id_address = "/api/get_text_histories_by_text_id"
                this.get_notification_histories_by_notification_id_address = "/api/get_gkill_notification_histories_by_notification_id"
                this.get_application_config_address = "/api/get_application_config"
                this.get_server_configs_address = "/api/get_server_configs"
                this.upload_files_address = "/api/upload_files"
                this.upload_gpslog_files_address = "/api/upload_gpslog_files"
                this.update_application_config_address = "/api/update_application_config"
                this.update_tag_struct_address = "/api/update_tag_struct"
                this.update_rep_struct_address = "/api/update_rep_struct"
                this.update_device_struct_address = "/api/update_device_struct"
                this.update_rep_type_struct_address = "/api/update_rep_type_struct"
                this.update_kftl_template_address = "/api/update_kftl_template"
                this.update_account_status_address = "/api/update_account_status"
                this.update_user_reps_address = "/api/update_user_reps"
                this.update_server_configs_address = "/api/update_server_configs"
                this.add_account_address = "/api/add_user"
                this.generate_tls_file_address = "/api/generate_tls_file"
                this.get_gps_log_address = "/api/get_gps_log"
                this.get_kftl_template_address = "/api/get_kftl_template"
                this.get_gkill_info_address = "/api/get_gkill_info"
                this.get_share_mi_task_list_infos_address = "/api/get_share_mi_task_list_infos"
                this.add_share_mi_task_list_info_address = "/api/add_share_mi_task_list_info"
                this.update_share_mi_task_list_info_address = "/api/update_share_mi_task_list_info"
                this.delete_share_mi_task_list_infos_address = "/api/delete_share_mi_task_list_infos"
                this.get_mi_shared_tasks_address = "/api/get_mi_shared_tasks"
                this.get_repositories_address = "/api/get_repositories"
                this.get_gkill_notification_public_key_address = "/api/get_gkill_notification_public_key"
                this.register_gkill_notification_address = "/api/register_gkill_notification"
                this.gkill_webpush_service_worker_js_address = "/serviceWorker.js"
                this.open_directory_address = "/api/open_directory"
                this.open_file_address = "/api/open_file"
                this.reload_repositories_address = "/api/reload_repositories"
                this.login_method = "POST"
                this.logout_method = "POST"
                this.reset_password_method = "POST"
                this.set_new_password_method = "POST"
                this.add_tag_method = "POST"
                this.add_text_method = "POST"
                this.add_notification_method = "POST"
                this.add_kmemo_method = "POST"
                this.add_urlog_method = "POST"
                this.add_nlog_method = "POST"
                this.add_timeis_method = "POST"
                this.add_mi_method = "POST"
                this.add_lantana_method = "POST"
                this.add_rekyou_method = "POST"
                this.update_tag_method = "POST"
                this.update_text_method = "POST"
                this.update_notification_method = "POST"
                this.update_kmemo_method = "POST"
                this.update_nlog_method = "POST"
                this.update_urlog_method = "POST"
                this.update_timeis_method = "POST"
                this.update_mi_method = "POST"
                this.update_lantana_method = "POST"
                this.update_idf_kyou_method = "POST"
                this.update_rekyou_method = "POST"
                this.get_kyous_method = "POST"
                this.get_kyou_method = "POST"
                this.get_kmemo_method = "POST"
                this.get_urlog_method = "POST"
                this.get_nlog_method = "POST"
                this.get_timeis_method = "POST"
                this.get_mi_method = "POST"
                this.get_lantana_method = "POST"
                this.get_git_commit_log_method = "POST"
                this.get_idf_kyou_method = "POST"
                this.get_rekyou_method = "POST"
                this.get_rekyous_method = "POST"
                this.get_git_commit_logs_method = "POST"
                this.get_mi_board_list_method = "POST"
                this.get_plaing_timeis_method = "POST"
                this.get_all_tag_names_method = "POST"
                this.get_all_rep_names_method = "POST"
                this.get_tags_by_target_id_method = "POST"
                this.get_tag_histories_by_tag_id_method = "POST"
                this.get_texts_by_target_id_method = "POST"
                this.get_notifications_by_target_id_method = "POST"
                this.get_text_histories_by_text_id_method = "POST"
                this.get_notification_histories_by_notification_id_method = "POST"
                this.get_application_config_method = "POST"
                this.get_server_configs_method = "POST"
                this.upload_files_method = "POST"
                this.upload_gpslog_files_method = "POST"
                this.update_application_config_method = "POST"
                this.update_tag_struct_method = "POST"
                this.update_rep_struct_method = "POST"
                this.update_device_struct_method = "POST"
                this.update_rep_type_struct_method = "POST"
                this.update_kftl_template_method = "POST"
                this.update_account_status_method = "POST"
                this.update_user_reps_method = "POST"
                this.update_server_config_method = "POST"
                this.add_account_method = "POST"
                this.generate_tls_file_method = "POST"
                this.get_gps_log_method = "POST"
                this.get_kftl_template_method = "POST"
                this.get_gkill_info_method = "POST"
                this.get_share_mi_task_list_infos_method = "POST"
                this.add_share_mi_task_list_info_method = "POST"
                this.update_share_mi_task_list_info_method = "POST"
                this.delete_share_mi_task_list_infos_method = "POST"
                this.get_mi_shared_tasks_method = "POST"
                this.get_repositories_method = "POST"
                this.get_gkill_notification_public_key_method = "POST"
                this.register_gkill_notification_method = "POST"
                this.open_directory_method = "POST"
                this.open_file_method = "POST"
                this.reload_repositories_method = "POST"
        }

        async login(req: LoginRequest): Promise<LoginResponse> {
                const res = await fetch(this.login_address, {
                        'method': this.login_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                })
                const json = await res.json()
                const response: LoginResponse = json
                this.check_auth(response)
                return response
        }

        async logout(req: LogoutRequest): Promise<LogoutResponse> {
                const res = await fetch(this.logout_address, {
                        'method': this.logout_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: LogoutResponse = json
                return response
        }

        async reset_password(req: ResetPasswordRequest): Promise<ResetPasswordResponse> {
                const res = await fetch(this.reset_password_address, {
                        'method': this.reset_password_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: ResetPasswordResponse = json
                this.check_auth(response)
                return response
        }

        async set_new_password(req: SetNewPasswordRequest): Promise<SetNewPasswordResponse> {
                const res = await fetch(this.set_new_password_address, {
                        'method': this.set_new_password_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: SetNewPasswordResponse = json
                this.check_auth(response)
                return response
        }

        async add_tag(req: AddTagRequest): Promise<AddTagResponse> {
                const res = await fetch(this.add_tag_address, {
                        'method': this.add_tag_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddTagResponse = json
                this.check_auth(response)
                return response
        }

        async add_text(req: AddTextRequest): Promise<AddTextResponse> {
                const res = await fetch(this.add_text_address, {
                        'method': this.add_text_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddTextResponse = json
                this.check_auth(response)
                return response
        }

        async add_notification(req: AddNotificationRequest): Promise<AddNotificationResponse> {
                const res = await fetch(this.add_notification_address, {
                        'method': this.add_notification_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddNotificationResponse = json
                this.check_auth(response)
                return response
        }

        async add_kmemo(req: AddKmemoRequest): Promise<AddKmemoResponse> {
                const res = await fetch(this.add_kmemo_address, {
                        'method': this.add_kmemo_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddKmemoResponse = json
                this.check_auth(response)
                return response
        }

        async add_urlog(req: AddURLogRequest): Promise<AddURLogResponse> {
                const res = await fetch(this.add_urlog_address, {
                        'method': this.add_urlog_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddURLogResponse = json
                this.check_auth(response)
                return response
        }

        async add_nlog(req: AddNlogRequest): Promise<AddNlogResponse> {
                const res = await fetch(this.add_nlog_address, {
                        'method': this.add_nlog_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddNlogResponse = json
                this.check_auth(response)
                return response
        }

        async add_timeis(req: AddTimeisRequest): Promise<AddTimeisResponse> {
                const res = await fetch(this.add_timeis_address, {
                        'method': this.add_timeis_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddTimeisResponse = json
                this.check_auth(response)
                return response
        }

        async add_mi(req: AddMiRequest): Promise<AddMiResponse> {
                const res = await fetch(this.add_mi_address, {
                        'method': this.add_mi_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddMiResponse = json
                this.check_auth(response)
                return response
        }

        async add_lantana(req: AddLantanaRequest): Promise<AddLantanaResponse> {
                const res = await fetch(this.add_lantana_address, {
                        'method': this.add_lantana_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddLantanaResponse = json
                this.check_auth(response)
                return response
        }

        async add_rekyou(req: AddReKyouRequest): Promise<AddReKyouResponse> {
                const res = await fetch(this.add_rekyou_address, {
                        'method': this.add_rekyou_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddReKyouResponse = json
                this.check_auth(response)
                return response
        }

        async update_tag(req: UpdateTagRequest): Promise<UpdateTagResponse> {
                const res = await fetch(this.update_tag_address, {
                        'method': this.update_tag_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateTagResponse = json
                this.check_auth(response)
                return response
        }

        async update_text(req: UpdateTextRequest): Promise<UpdateTextResponse> {
                const res = await fetch(this.update_text_address, {
                        'method': this.update_text_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateTextResponse = json
                this.check_auth(response)
                return response
        }

        async update_notification(req: UpdateNotificationRequest): Promise<UpdateNotificationResponse> {
                const res = await fetch(this.update_notification_address, {
                        'method': this.update_notification_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateNotificationResponse = json
                this.check_auth(response)
                return response
        }

        async update_kmemo(req: UpdateKmemoRequest): Promise<UpdateKmemoResponse> {
                const res = await fetch(this.update_kmemo_address, {
                        'method': this.update_kmemo_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateKmemoResponse = json
                this.check_auth(response)
                return response
        }

        async update_urlog(req: UpdateURLogRequest): Promise<UpdateURLogResponse> {
                const res = await fetch(this.update_urlog_address, {
                        'method': this.update_urlog_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateURLogResponse = json
                this.check_auth(response)
                return response
        }

        async update_nlog(req: UpdateNlogRequest): Promise<UpdateNlogResponse> {
                const res = await fetch(this.update_nlog_address, {
                        'method': this.update_nlog_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateNlogResponse = json
                this.check_auth(response)
                return response
        }

        async update_timeis(req: UpdateTimeisRequest): Promise<UpdateTimeisResponse> {
                const res = await fetch(this.update_timeis_address, {
                        'method': this.update_timeis_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateTimeisResponse = json
                this.check_auth(response)
                return response
        }

        async update_mi(req: UpdateMiRequest): Promise<UpdateMiResponse> {
                const res = await fetch(this.update_mi_address, {
                        'method': this.update_mi_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateMiResponse = json
                this.check_auth(response)
                return response
        }

        async update_lantana(req: UpdateLantanaRequest): Promise<UpdateLantanaResponse> {
                const res = await fetch(this.update_lantana_address, {
                        'method': this.update_lantana_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateLantanaResponse = json
                this.check_auth(response)
                return response
        }

        async update_idf_kyou(req: UpdateIDFKyouRequest): Promise<UpdateIDFKyouResponse> {
                const res = await fetch(this.update_idf_kyou_address, {
                        'method': this.update_idf_kyou_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateIDFKyouResponse = json
                this.check_auth(response)
                return response
        }

        async update_rekyou(req: UpdateReKyouRequest): Promise<UpdateReKyouResponse> {
                const res = await fetch(this.update_rekyou_address, {
                        'method': this.update_rekyou_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateReKyouResponse = json
                this.check_auth(response)
                return response
        }

        async get_kyous(req: GetKyousRequest): Promise<GetKyousResponse> {
                const res = await fetch(this.get_kyous_address, {
                        'method': this.get_kyous_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetKyousResponse = json
                this.check_auth(response)
                if (!response.kyous) {
                        response.kyous = new Array<Kyou>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したKyouリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.kyous.length; i++) {
                        const kyou = new Kyou()
                        for (const key in response.kyous[i]) {
                                (kyou as any)[key] = (response.kyous[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (kyou as any)[key]) {
                                        (kyou as any)[key] = moment((kyou as any)[key]).toDate()
                                }
                        }
                        response.kyous[i] = kyou
                }

                return response
        }

        async get_kyou(req: GetKyouRequest): Promise<GetKyouResponse> {
                const res = await fetch(this.get_kyou_address, {
                        'method': this.get_kyou_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetKyouResponse = new GetKyouResponse()
                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したKyouリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.kyou_histories.length; i++) {
                        const kyou = new Kyou()
                        for (const key in response.kyou_histories[i]) {
                                (kyou as any)[key] = (response.kyou_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (kyou as any)[key]) {
                                        (kyou as any)[key] = moment((kyou as any)[key]).toDate()
                                }
                        }
                        response.kyou_histories[i] = kyou
                }
                this.check_auth(response)
                return response
        }

        async get_kmemo(req: GetKmemoRequest): Promise<GetKmemoResponse> {
                const res = await fetch(this.get_kmemo_address, {
                        'method': this.get_kmemo_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetKmemoResponse = json
                if (!response.kmemo_histories) {
                        response.kmemo_histories = new Array<Kmemo>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.kmemo_histories.length; i++) {
                        const kmemo = new Kmemo()
                        for (const key in response.kmemo_histories[i]) {
                                (kmemo as any)[key] = (response.kmemo_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (kmemo as any)[key]) {
                                        (kmemo as any)[key] = moment((kmemo as any)[key]).toDate()
                                }
                        }
                        response.kmemo_histories[i] = kmemo
                }

                this.check_auth(response)
                return response
        }

        async get_urlog(req: GetURLogRequest): Promise<GetURLogResponse> {
                const res = await fetch(this.get_urlog_address, {
                        'method': this.get_urlog_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetURLogResponse = json
                if (!response.urlog_histories) {
                        response.urlog_histories = new Array<URLog>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.urlog_histories.length; i++) {
                        const urlog = new URLog()
                        for (const key in response.urlog_histories[i]) {
                                (urlog as any)[key] = (response.urlog_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (urlog as any)[key]) {
                                        (urlog as any)[key] = moment((urlog as any)[key]).toDate()
                                }
                        }
                        response.urlog_histories[i] = urlog
                }

                this.check_auth(response)
                return response
        }

        async get_nlog(req: GetNlogRequest): Promise<GetNlogResponse> {
                const res = await fetch(this.get_nlog_address, {
                        'method': this.get_nlog_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetNlogResponse = json
                if (!response.nlog_histories) {
                        response.nlog_histories = new Array<Nlog>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.nlog_histories.length; i++) {
                        const nlog = new Nlog()
                        for (const key in response.nlog_histories[i]) {
                                (nlog as any)[key] = (response.nlog_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (nlog as any)[key]) {
                                        (nlog as any)[key] = moment((nlog as any)[key]).toDate()
                                }
                        }
                        response.nlog_histories[i] = nlog
                }

                this.check_auth(response)
                return response
        }

        async get_timeis(req: GetTimeisRequest): Promise<GetTimeisResponse> {
                const res = await fetch(this.get_timeis_address, {
                        'method': this.get_timeis_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetTimeisResponse = json
                if (!response.timeis_histories) {
                        response.timeis_histories = new Array<TimeIs>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.timeis_histories.length; i++) {
                        const timeis = new TimeIs()
                        for (const key in response.timeis_histories[i]) {
                                (timeis as any)[key] = (response.timeis_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (timeis as any)[key]) {
                                        (timeis as any)[key] = moment((timeis as any)[key]).toDate()
                                }
                        }
                        response.timeis_histories[i] = timeis
                }

                this.check_auth(response)
                return response
        }

        async get_mi(req: GetMiRequest): Promise<GetMiResponse> {
                const res = await fetch(this.get_mi_address, {
                        'method': this.get_mi_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetMiResponse = json
                if (!response.mi_histories) {
                        response.mi_histories = new Array<Mi>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.mi_histories.length; i++) {
                        const mis = new Mi()
                        for (const key in response.mi_histories[i]) {
                                (mis as any)[key] = (response.mi_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (mis as any)[key]) {
                                        (mis as any)[key] = moment((mis as any)[key]).toDate()
                                }
                        }
                        response.mi_histories[i] = mis
                }

                this.check_auth(response)
                return response
        }

        async get_lantana(req: GetLantanaRequest): Promise<GetLantanaResponse> {
                const res = await fetch(this.get_lantana_address, {
                        'method': this.get_lantana_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetLantanaResponse = json
                if (!response.lantana_histories) {
                        response.lantana_histories = new Array<Lantana>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.lantana_histories.length; i++) {
                        const lantanas = new Lantana()
                        for (const key in response.lantana_histories[i]) {
                                (lantanas as any)[key] = (response.lantana_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (lantanas as any)[key]) {
                                        (lantanas as any)[key] = moment((lantanas as any)[key]).toDate()
                                }
                        }
                        response.lantana_histories[i] = lantanas
                }

                this.check_auth(response)
                return response
        }

        async get_rekyou(req: GetReKyouRequest): Promise<GetReKyouResponse> {
                const res = await fetch(this.get_rekyou_address, {
                        'method': this.get_rekyou_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetReKyouResponse = json
                if (!response.rekyou_histories) {
                        response.rekyou_histories = new Array<ReKyou>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.rekyou_histories.length; i++) {
                        const rekyou = new ReKyou()
                        for (const key in response.rekyou_histories[i]) {
                                (rekyou as any)[key] = (response.rekyou_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (rekyou as any)[key]) {
                                        (rekyou as any)[key] = moment((rekyou as any)[key]).toDate()
                                }
                        }
                        response.rekyou_histories[i] = rekyou
                }

                this.check_auth(response)
                return response
        }

        async get_git_commit_log(req: GetGitCommitLogRequest): Promise<GetGitCommitLogResponse> {
                const res = await fetch(this.get_git_commit_log_address, {
                        'method': this.get_git_commit_log_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetGitCommitLogResponse = json
                if (!response.git_commit_log_histories) {
                        response.git_commit_log_histories = new Array<GitCommitLog>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.git_commit_log_histories.length; i++) {
                        const git_commit_log = new GitCommitLog()
                        for (const key in response.git_commit_log_histories[i]) {
                                (git_commit_log as any)[key] = (response.git_commit_log_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (git_commit_log as any)[key]) {
                                        (git_commit_log as any)[key] = moment((git_commit_log as any)[key]).toDate()
                                }
                        }
                        response.git_commit_log_histories[i] = git_commit_log
                }

                this.check_auth(response)
                return response
        }

        async get_idf_kyou(req: GetIDFKyouRequest): Promise<GetIDFKyouResponse> {
                const res = await fetch(this.get_idf_kyou_address, {
                        'method': this.get_idf_kyou_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetIDFKyouResponse = json
                if (!response.idf_kyou_histories) {
                        response.idf_kyou_histories = new Array<IDFKyou>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.idf_kyou_histories.length; i++) {
                        const idf_kyou = new IDFKyou()
                        for (const key in response.idf_kyou_histories[i]) {
                                (idf_kyou as any)[key] = (response.idf_kyou_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (idf_kyou as any)[key]) {
                                        (idf_kyou as any)[key] = moment((idf_kyou as any)[key]).toDate()
                                }
                        }
                        response.idf_kyou_histories[i] = idf_kyou
                }

                this.check_auth(response)
                return response
        }

        async get_mi_board_list(req: GetMiBoardRequest): Promise<GetMiBoardResponse> {
                const res = await fetch(this.get_mi_board_list_address, {
                        'method': this.get_mi_board_list_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetMiBoardResponse = json
                this.check_auth(response)
                return response
        }

        async get_plaing_timeis(req: GetPlaingTimeisRequest): Promise<GetPlaingTimeisResponse> {
                const res = await fetch(this.get_plaing_timeis_address, {
                        'method': this.get_plaing_timeis_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetPlaingTimeisResponse = json
                this.check_auth(response)
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                this.check_auth(response)
                if (!response.plaing_timeis_kyous) {
                        response.plaing_timeis_kyous = new Array<Kyou>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したKyouリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.plaing_timeis_kyous.length; i++) {
                        const kyou = new Kyou()
                        for (const key in response.plaing_timeis_kyous[i]) {
                                (kyou as any)[key] = (response.plaing_timeis_kyous[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (kyou as any)[key]) {
                                        (kyou as any)[key] = moment((kyou as any)[key]).toDate()
                                }
                        }
                        response.plaing_timeis_kyous[i] = kyou
                }
                return response
        }

        async get_all_tag_names(req: GetAllTagNamesRequest): Promise<GetAllTagNamesResponse> {
                const res = await fetch(this.get_all_tag_names_address, {
                        'method': this.get_all_tag_names_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetAllTagNamesResponse = json
                this.check_auth(response)
                return response
        }

        async get_all_rep_names(req: GetAllRepNamesRequest): Promise<GetAllRepNamesResponse> {
                const res = await fetch(this.get_all_rep_names_address, {
                        'method': this.get_all_rep_names_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetAllRepNamesResponse = json
                this.check_auth(response)
                return response
        }

        async get_tags_by_target_id(req: GetTagsByTargetIDRequest): Promise<GetTagsByTargetIDResponse> {
                const res = await fetch(this.get_tags_by_target_id_address, {
                        'method': this.get_tags_by_target_id_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetTagsByTargetIDResponse = json
                this.check_auth(response)
                if (!response.tags) {
                        response.tags = new Array<Tag>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したTagリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.tags.length; i++) {
                        const tag = new Tag()
                        for (const key in response.tags[i]) {
                                (tag as any)[key] = (response.tags[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (tag as any)[key]) {
                                        (tag as any)[key] = moment((tag as any)[key]).toDate()
                                }
                        }
                        response.tags[i] = tag
                }
                this.check_auth(response)
                return response
        }

        async get_tag_histories_by_tag_id(req: GetTagHistoryByTagIDRequest): Promise<GetTagHistoryByTagIDResponse> {
                const res = await fetch(this.get_tag_histories_by_tag_id_address, {
                        'method': this.get_tag_histories_by_tag_id_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetTagHistoryByTagIDResponse = json
                this.check_auth(response)
                if (!response.tag_histories) {
                        response.tag_histories = new Array<Tag>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したTagリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.tag_histories.length; i++) {
                        const tag = new Tag()
                        for (const key in response.tag_histories[i]) {
                                (tag as any)[key] = (response.tag_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (tag as any)[key]) {
                                        (tag as any)[key] = moment((tag as any)[key]).toDate()
                                }
                        }
                        response.tag_histories[i] = tag
                }
                this.check_auth(response)
                return response
        }

        async get_texts_by_target_id(req: GetTextsByTargetIDRequest): Promise<GetTextsByTargetIDResponse> {
                const res = await fetch(this.get_texts_by_target_id_address, {
                        'method': this.get_texts_by_target_id_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetTextsByTargetIDResponse = json
                this.check_auth(response)
                if (!response.texts) {
                        response.texts = new Array<Text>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したTextリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.texts.length; i++) {
                        const text = new Text()
                        for (const key in response.texts[i]) {
                                (text as any)[key] = (response.texts[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (text as any)[key]) {
                                        (text as any)[key] = moment((text as any)[key]).toDate()
                                }
                        }
                        response.texts[i] = text
                }
                this.check_auth(response)
                return response
        }

        async get_notifications_by_target_id(req: GetNotificationsByTargetIDRequest): Promise<GetNotificationsByTargetIDResponse> {
                const res = await fetch(this.get_notifications_by_target_id_address, {
                        'method': this.get_notifications_by_target_id_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetNotificationsByTargetIDResponse = json
                this.check_auth(response)
                if (!response.notifications) {
                        response.notifications = new Array<Notification>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したNotificationリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.notifications.length; i++) {
                        const notification = new Notification()
                        for (const key in response.notifications[i]) {
                                (notification as any)[key] = (response.notifications[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (notification as any)[key]) {
                                        (notification as any)[key] = moment((notification as any)[key]).toDate()
                                }
                        }
                        response.notifications[i] = notification
                }
                this.check_auth(response)
                return response
        }

        async get_text_history_by_text_id(req: GetTextHistoryByTextIDRequest): Promise<GetTextHistoryByTextIDResponse> {
                const res = await fetch(this.get_text_histories_by_text_id_address, {
                        'method': this.get_text_histories_by_text_id_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetTextHistoryByTextIDResponse = json
                this.check_auth(response)
                if (!response.text_histories) {
                        response.text_histories = new Array<Text>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したTextリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.text_histories.length; i++) {
                        const text = new Text()
                        for (const key in response.text_histories[i]) {
                                (text as any)[key] = (response.text_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (text as any)[key]) {
                                        (text as any)[key] = moment((text as any)[key]).toDate()
                                }
                        }
                        response.text_histories[i] = text
                }
                this.check_auth(response)
                return response
        }

        async get_notification_history_by_notification_id(req: GetNotificationHistoryByNotificationIDRequest): Promise<GetNotificationHistoryByNotificationIDResponse> {
                const res = await fetch(this.get_notification_histories_by_notification_id_address, {
                        'method': this.get_notification_histories_by_notification_id_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetNotificationHistoryByNotificationIDResponse = json
                this.check_auth(response)
                if (!response.notification_histories) {
                        response.notification_histories = new Array<Notification>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したNotificationリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.notification_histories.length; i++) {
                        const notification = new Notification()
                        for (const key in response.notification_histories[i]) {
                                (notification as any)[key] = (response.notification_histories[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (notification as any)[key]) {
                                        (notification as any)[key] = moment((notification as any)[key]).toDate()
                                }
                        }
                        response.notification_histories[i] = notification
                }
                this.check_auth(response)
                return response
        }



        async get_application_config(req: GetApplicationConfigRequest): Promise<GetApplicationConfigResponse> {
                const res = await fetch(this.get_application_config_address, {
                        'method': this.get_application_config_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response = json as unknown as GetApplicationConfigResponse
                this.check_auth(response)

                const gkill_info_req = new GetGkillInfoRequest()
                const gkill_info_res = await this.get_gkill_info(gkill_info_req)
                if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
                        response.errors = response.errors.concat(gkill_info_res.errors)
                        this.check_auth(response)
                        return response
                }
                if (gkill_info_res.messages && gkill_info_res.messages.length !== 0) {
                        // response.messages = response.messages.concat(gkill_info_res.messages)
                }

                const application_config = new ApplicationConfig()
                application_config.tag_struct = response.application_config.tag_struct ? response.application_config.tag_struct : []
                application_config.device_struct = response.application_config.device_struct ? response.application_config.device_struct : []
                application_config.rep_struct = response.application_config.rep_struct ? response.application_config.rep_struct : []
                application_config.rep_type_struct = response.application_config.rep_type_struct ? response.application_config.rep_type_struct : []
                application_config.kftl_template_struct = response.application_config.kftl_template_struct ? response.application_config.kftl_template_struct : []
                application_config.account_is_admin = response.application_config.account_is_admin
                application_config.device = response.application_config.device
                application_config.enable_browser_cache = response.application_config.enable_browser_cache
                application_config.google_map_api_key = response.application_config.google_map_api_key
                application_config.is_loaded = response.application_config.is_loaded
                application_config.mi_default_board = response.application_config.mi_default_board
                application_config.rykv_hot_reload = response.application_config.rykv_hot_reload
                application_config.rykv_image_list_column_number = response.application_config.rykv_image_list_column_number
                application_config.user_id = response.application_config.user_id
                application_config.session_is_local = response.application_config.session_is_local
                response.application_config = application_config

                await response.application_config.load_all()
                this.check_auth(response)

                this.set_google_map_api_key(response.application_config.google_map_api_key)
                return response
        }

        async get_server_configs(req: GetServerConfigsRequest): Promise<GetServerConfigsResponse> {
                const res = await fetch(this.get_server_configs_address, {
                        'method': this.get_server_configs_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetServerConfigsResponse = json
                this.check_auth(response)
                return response
        }

        async upload_files(req: UploadFilesRequest): Promise<UploadFilesResponse> {
                const res = await fetch(this.upload_files_address, {
                        'method': this.upload_files_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UploadFilesResponse = json
                this.check_auth(response)
                return response
        }

        async upload_gpslog_files(req: UploadGPSLogFilesRequest): Promise<UploadGPSLogFilesResponse> {
                const res = await fetch(this.upload_gpslog_files_address, {
                        'method': this.upload_gpslog_files_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UploadGPSLogFilesResponse = json
                this.check_auth(response)
                return response
        }

        async update_application_config(req: UpdateApplicationConfigRequest): Promise<UpdateApplicationConfigResponse> {
                const res = await fetch(this.update_application_config_address, {
                        'method': this.update_application_config_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateApplicationConfigResponse = json
                this.check_auth(response)
                return response
        }

        async update_tag_struct(req: UpdateTagStructRequest): Promise<UpdateTagStructResponse> {
                const res = await fetch(this.update_tag_struct_address, {
                        'method': this.update_tag_struct_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateTagStructResponse = json
                this.check_auth(response)
                return response
        }

        async update_rep_struct(req: UpdateRepStructRequest): Promise<UpdateRepStructResponse> {
                const res = await fetch(this.update_rep_struct_address, {
                        'method': this.update_rep_struct_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateRepStructResponse = json
                this.check_auth(response)
                return response
        }

        async update_device_struct(req: UpdateDeviceStructRequest): Promise<UpdateDeviceStructResponse> {
                const res = await fetch(this.update_device_struct_address, {
                        'method': this.update_device_struct_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateDeviceStructResponse = json
                this.check_auth(response)
                return response
        }

        async update_rep_type_struct(req: UpdateRepTypeStructRequest): Promise<UpdateRepTypeStructResponse> {
                const res = await fetch(this.update_rep_type_struct_address, {
                        'method': this.update_rep_type_struct_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateRepTypeStructResponse = json
                this.check_auth(response)
                return response
        }

        async update_kftl_template(req: UpdateKFTLTemplateRequest): Promise<UpdateKFTLTemplateResponse> {
                const res = await fetch(this.update_kftl_template_address, {
                        'method': this.update_kftl_template_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateKFTLTemplateResponse = json
                this.check_auth(response)
                return response
        }

        async update_account_status(req: UpdateAccountStatusRequest): Promise<UpdateAccountStatusResponse> {
                const res = await fetch(this.update_account_status_address, {
                        'method': this.update_account_status_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateAccountStatusResponse = json
                this.check_auth(response)
                return response
        }

        async update_user_reps(req: UpdateUserRepsRequest): Promise<UpdateUserRepsResponse> {
                const res = await fetch(this.update_user_reps_address, {
                        'method': this.update_user_reps_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateUserRepsResponse = json
                this.check_auth(response)
                return response
        }

        async update_server_config(req: UpdateServerConfigsRequest): Promise<UpdateServerConfigsResponse> {
                const res = await fetch(this.update_server_configs_address, {
                        'method': this.update_server_config_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: UpdateServerConfigsResponse = json
                this.check_auth(response)
                return response
        }

        async add_account(req: AddAccountRequest): Promise<AddAccountResponse> {
                const res = await fetch(this.add_account_address, {
                        'method': this.add_account_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: AddAccountResponse = json
                this.check_auth(response)
                return response
        }

        async generate_tls_file(req: GenerateTLSFileRequest): Promise<GenerateTLSFileResponse> {
                const res = await fetch(this.generate_tls_file_address, {
                        'method': this.generate_tls_file_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GenerateTLSFileResponse = json
                this.check_auth(response)
                return response
        }

        async get_gps_log(req: GetGPSLogRequest): Promise<GetGPSLogResponse> {
                const res = await fetch(this.get_gps_log_address, {
                        'method': this.get_gps_log_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })

                const json = await res.json()

                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetGPSLogResponse = json
                if (!response.gps_logs) {
                        response.gps_logs = new Array<GPSLog>()
                }

                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                // 取得したリストの型変換（そのままキャストするとメソッドが生えないため）
                for (let i = 0; i < response.gps_logs.length; i++) {
                        const gpslog = new GPSLog()
                        for (const key in response.gps_logs[i]) {
                                (gpslog as any)[key] = (response.gps_logs[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (gpslog as any)[key]) {
                                        (gpslog as any)[key] = moment((gpslog as any)[key]).toDate()
                                }
                        }
                        response.gps_logs[i] = gpslog
                }
                this.check_auth(response)
                return response
        }

        async get_kftl_templates(req: GetKFTLTemplatesRequest): Promise<GetKFTLTemplatesResponse> {
                const res = await fetch(this.get_kftl_template_address, {
                        'method': this.get_kftl_template_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetKFTLTemplatesResponse = json
                this.check_auth(response)
                return response
        }

        async get_gkill_info(req: GetGkillInfoRequest): Promise<GetGkillInfoResponse> {
                const res = await fetch(this.get_gkill_info_address, {
                        'method': this.get_gkill_info_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetGkillInfoResponse = json
                this.check_auth(response)
                return response
        }

        async get_share_mi_task_list_infos(req: GetShareMiTaskListInfosRequest): Promise<GetShareMiTaskListInfosResponse> {
                const res = await fetch(this.get_share_mi_task_list_infos_address, {
                        'method': this.get_share_mi_task_list_infos_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: GetShareMiTaskListInfosResponse = new GetShareMiTaskListInfosResponse()
                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                for (let i = 0; i < response.share_mi_task_list_infos.length; i++) {
                        const share_mi_task_list_info = new ShareMiTaskListInfo()
                        for (const key in json.share_mi_task_list_infos[i]) {
                                (share_mi_task_list_info as any)[key] = json.share_mi_task_list_infos[i][key]
                        }
                        response.share_mi_task_list_infos[i] = share_mi_task_list_info
                }
                this.check_auth(response)
                return response
        }

        async add_share_mi_task_list_info(req: AddShareMiTaskListInfoRequest): Promise<AddShareMiTaskListInfoResponse> {
                const res = await fetch(this.add_share_mi_task_list_info_address, {
                        'method': this.add_share_mi_task_list_info_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: AddShareMiTaskListInfoResponse = new AddShareMiTaskListInfoResponse()
                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                const share_mi_task_list_info = new ShareMiTaskListInfo()
                for (const key in json.share_mi_task_list_info) {
                        (share_mi_task_list_info as any)[key] = json.share_mi_task_list_info[key]
                }
                response.share_mi_task_list_info = share_mi_task_list_info
                this.check_auth(response)
                return response
        }

        async update_share_mi_task_list_info(req: UpdateShareMiTaskListInfoRequest): Promise<UpdateShareMiTaskListInfoResponse> {
                const res = await fetch(this.update_share_mi_task_list_info_address, {
                        'method': this.update_share_mi_task_list_info_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: UpdateShareMiTaskListInfoResponse = new UpdateShareMiTaskListInfoResponse()
                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                const share_mi_task_list_info = new ShareMiTaskListInfo()
                for (const key in json.share_mi_task_list_info) {
                        (share_mi_task_list_info as any)[key] = json.share_mi_task_list_info[key]
                }
                response.share_mi_task_list_info = share_mi_task_list_info
                this.check_auth(response)
                return response
        }

        async delete_share_mi_task_list_infos(req: DeleteShareMiTaskListInfosRequest): Promise<DeleteShareMiTaskListInfosResponse> {
                const res = await fetch(this.delete_share_mi_task_list_infos_address, {
                        'method': this.delete_share_mi_task_list_infos_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                // Response型に合わせる（そのままキャストするとメソッドが生えないため）
                const response: DeleteShareMiTaskListInfosResponse = new DeleteShareMiTaskListInfosResponse()
                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                this.check_auth(response)
                return response
        }

        async get_mi_shared_tasks(req: GetSharedMiTasksRequest): Promise<GetSharedMiTasksResponse> {
                const res = await fetch(this.get_mi_shared_tasks_address, {
                        'method': this.get_mi_shared_tasks_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetSharedMiTasksResponse = new GetSharedMiTasksResponse()
                for (const key in json) {
                        (response as any)[key] = json[key]
                }
                for (let i = 0; i < json.mi_kyous.length; i++) {
                        const kyou = new Kyou()
                        for (const key in json.mi_kyous[i]) {
                                (kyou as any)[key] = (json.mi_kyous[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (kyou as any)[key]) {
                                        (kyou as any)[key] = moment((kyou as any)[key]).toDate()
                                }
                        }
                        response.mi_kyous[i] = kyou
                }
                for (let i = 0; i < json.mis.length; i++) {
                        const mi = new Mi()
                        for (const key in json.mis[i]) {
                                (mi as any)[key] = (json.mis[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (mi as any)[key]) {
                                        (mi as any)[key] = moment((mi as any)[key]).toDate()
                                }
                        }
                        response.mis[i] = mi
                }
                for (let i = 0; i < json.tags.length; i++) {
                        const tag = new Tag()
                        for (const key in json.tags[i]) {
                                (tag as any)[key] = (json.tags[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (tag as any)[key]) {
                                        (tag as any)[key] = moment((tag as any)[key]).toDate()
                                }
                        }
                        response.tags[i] = tag
                }
                for (let i = 0; i < json.texts.length; i++) {
                        const text = new Text()
                        for (const key in json.texts[i]) {
                                (text as any)[key] = (json.texts[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (text as any)[key]) {
                                        (text as any)[key] = moment((text as any)[key]).toDate()
                                }
                        }
                        response.texts[i] = text
                }
                for (let i = 0; i < json.timeiss.length; i++) {
                        const timeis = new TimeIs()
                        for (const key in json.timeiss[i]) {
                                (timeis as any)[key] = (json.timeiss[i] as any)[key]

                                // 時刻はDate型に変換
                                if (key.endsWith("time") && (timeis as any)[key]) {
                                        (timeis as any)[key] = moment((timeis as any)[key]).toDate()
                                }
                        }
                        response.timeiss[i] = timeis
                }
                return response
        }

        async get_repositories(req: GetRepositoriesRequest): Promise<GetRepositoriesResponse> {
                const res = await fetch(this.get_repositories_address, {
                        'method': this.get_repositories_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetRepositoriesResponse = json
                this.check_auth(response)
                return response
        }

        async get_gkill_notification_public_key(req: GetGkillNotificationPublicKeyRequest): Promise<GetGkillNotificationPublicKeyResponse> {
                const res = await fetch(this.get_gkill_notification_public_key_address, {
                        'method': this.get_gkill_notification_public_key_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: GetGkillNotificationPublicKeyResponse = json
                this.check_auth(response)
                return response
        }

        async register_gkill_notification(req: RegisterGkillNotificationRequest): Promise<RegisterGkillNotificationResponse> {
                const res = await fetch(this.register_gkill_notification_address, {
                        'method': this.register_gkill_notification_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: RegisterGkillNotificationResponse = json
                this.check_auth(response)
                return response
        }

        async open_directory(req: OpenDirectoryRequest): Promise<OpenDirectoryResponse> {
                const res = await fetch(this.open_directory_address, {
                        'method': this.open_directory_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: OpenDirectoryResponse = json
                this.check_auth(response)
                return response
        }

        async open_file(req: OpenFileRequest): Promise<OpenFileResponse> {
                const res = await fetch(this.open_file_address, {
                        'method': this.open_file_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: OpenFileResponse = json
                this.check_auth(response)
                return response
        }

        async reload_repositories(req: ReloadRepositoriesRequest): Promise<ReloadRepositoriesResponse> {
                const res = await fetch(this.reload_repositories_address, {
                        'method': this.reload_repositories_method,
                        headers: {
                                'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(req),
                        signal: req.abort_controller?.signal,
                })
                const json = await res.json()
                const response: ReloadRepositoriesResponse = json
                this.check_auth(response)
                return response
        }

        private gkill_session_id_cookie_key = "gkill_session_id"

        get_session_id(): string {
                const cookies = document.cookie.split(';')
                let session_id = cookies.find(
                        (cookie) => cookie.split('=')[0].trim() === this.gkill_session_id_cookie_key.trim()
                )?.replace(this.gkill_session_id_cookie_key + "=", "").trim()

                if (!session_id) {
                        this.set_session_id("")
                        session_id = ""
                }
                return session_id
        }

        set_session_id(session_id: string): void {
                if (session_id === "") {
                        document.cookie = this.gkill_session_id_cookie_key + "=" + session_id + "; max-age=0"
                } else {
                        document.cookie = this.gkill_session_id_cookie_key + "=" + session_id + "; max-age=" + 86400 * 400
                }
        }

        private google_map_api_key_cookie_key = "google_map_api_key"

        get_google_map_api_key(): string {
                const cookies = document.cookie.split(';')
                let google_map_api_key = cookies.find(
                        (cookie) => cookie.split('=')[0].trim() === this.google_map_api_key_cookie_key.trim()
                )?.replace(this.google_map_api_key_cookie_key + "=", "")

                if (!google_map_api_key) {
                        this.set_google_map_api_key("")
                        google_map_api_key = ""
                }
                return google_map_api_key
        }

        set_google_map_api_key(google_map_api_key: string): void {
                if (google_map_api_key === "") {
                        document.cookie = this.google_map_api_key_cookie_key + "=" + google_map_api_key + "; max-age=0"
                } else {
                        document.cookie = this.google_map_api_key_cookie_key + "=" + google_map_api_key + "; max-age=" + 86400 * 400
                }
        }

        generate_uuid(): string {
                const chars = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.split('')
                for (let i = 0, len = chars.length; i < len; i++) {
                        switch (chars[i]) {
                                case 'x':
                                        chars[i] = Math.floor(Math.random() * 16).toString(16)
                                        break
                                case 'y':
                                        chars[i] = (Math.floor(Math.random() * 4) + 8).toString(16)
                                        break
                        }
                }
                const uuid = chars.join('')
                return uuid
        }

        // 認証が通っていなかったらログイン画面に遷移する
        check_auth(res: GkillAPIResponse): void {
                if (res.errors && res.errors.length !== 0) {
                        res.errors.forEach(error => {
                                switch (error.error_code) {
                                        case "ERR000013": // AccountSessionNotFoundError
                                                this.set_session_id("")
                                                router.replace("/")
                                                break
                                        case "ERR000002": // AccountNotFoundError
                                                this.set_session_id("")
                                                router.replace("/")
                                                break
                                        case "ERR000238": // AccountDisabledError
                                                this.set_session_id("")
                                                router.replace("/")
                                                break
                                }
                        })
                }
        }

        private saved_application_config: ApplicationConfig | null
        set_saved_application_config(application_config: ApplicationConfig): void {
                this.saved_application_config = application_config
        }
        get_saved_application_config(): ApplicationConfig | null {
                return this.saved_application_config
        }

        private rykv_find_kyou_querys_localstorage_key = "rykv_find_kyou_querys"
        set_saved_rykv_find_kyou_querys(querys: Array<FindKyouQuery>): void {
                window.localStorage.setItem(this.rykv_find_kyou_querys_localstorage_key, JSON.stringify(querys))
        }
        get_saved_rykv_find_kyou_querys(): Array<FindKyouQuery> {
                const query_json_string = window.localStorage.getItem(this.rykv_find_kyou_querys_localstorage_key)
                if (!query_json_string) {
                        return new Array<FindKyouQuery>()
                }
                const querys_json: any = JSON.parse(query_json_string)
                if (!querys_json) {
                        return new Array<FindKyouQuery>()
                }
                // 型に合わせる（そのままキャストするとメソッドが生えないため）
                const querys = Array<FindKyouQuery>()
                for (let i = 0; i < querys_json.length; i++) {
                        const query = new FindKyouQuery()
                        for (const key in querys_json[i]) {
                                (query as any)[key] = querys_json[i][key]
                                if ((key.endsWith("time") || key.endsWith("date")) && (query as any)[key]) {
                                        (query as any)[key] = moment((query as any)[key]).toDate()
                                }
                        }
                        querys.push(query)
                }
                return querys
        }

        private mi_find_kyou_querys_localstorage_key = "mi_find_kyou_querys"
        set_saved_mi_find_kyou_querys(querys: Array<FindKyouQuery>): void {
                window.localStorage.setItem(this.mi_find_kyou_querys_localstorage_key, JSON.stringify(querys))
        }
        get_saved_mi_find_kyou_querys(): Array<FindKyouQuery> {
                const query_json_string = window.localStorage.getItem(this.mi_find_kyou_querys_localstorage_key)
                if (!query_json_string) {
                        return new Array<FindKyouQuery>()
                }
                const querys_json: any = JSON.parse(query_json_string)
                if (!querys_json) {
                        return new Array<FindKyouQuery>()
                }
                // 型に合わせる（そのままキャストするとメソッドが生えないため）
                const querys = Array<FindKyouQuery>()
                for (let i = 0; i < querys_json.length; i++) {
                        const query = new FindKyouQuery()
                        for (const key in querys_json[i]) {
                                (query as any)[key] = querys_json[i][key]
                                if ((key.endsWith("time") || key.endsWith("date")) && (query as any)[key]) {
                                        (query as any)[key] = moment((query as any)[key]).toDate()
                                }
                        }
                        querys.push(query)
                }
                return querys
        }

        private rykv_scroll_indexs_localstorage_key = "rykv_scroll_indexs"
        set_saved_rykv_scroll_indexs(indexs: Array<number>): void {
                window.localStorage.setItem(this.rykv_scroll_indexs_localstorage_key, JSON.stringify(indexs))
        }
        get_saved_rykv_scroll_indexs(): Array<number> {
                const indexs_json_string = window.localStorage.getItem(this.rykv_scroll_indexs_localstorage_key)
                if (!indexs_json_string) {
                        return new Array<number>()
                }
                const indexs_json: any = JSON.parse(indexs_json_string)
                if (!indexs_json) {
                        return new Array<number>()
                }
                const indexs: Array<number> = indexs_json
                return indexs
        }

        private mi_scroll_indexs_localstorage_key = "mi_scroll_indexs"
        set_saved_mi_scroll_indexs(indexs: Array<number>): void {
                window.localStorage.setItem(this.mi_scroll_indexs_localstorage_key, JSON.stringify(indexs))
        }
        get_saved_mi_scroll_indexs(): Array<number> {
                const indexs_json_string = window.localStorage.getItem(this.mi_scroll_indexs_localstorage_key)
                if (!indexs_json_string) {
                        return new Array<number>()
                }
                const indexs_json: any = JSON.parse(indexs_json_string)
                if (!indexs_json) {
                        return new Array<number>()
                }
                const indexs: Array<number> = indexs_json
                return indexs
        }
}

export class GkillAPIForSharedMi extends GkillAPI {
        private static gkill_api_for_shared_mi: GkillAPIForSharedMi = new GkillAPIForSharedMi()
        static get_instance_for_share_mi(): GkillAPIForSharedMi {
                return GkillAPIForSharedMi.gkill_api_for_shared_mi
        }

        // ここのデータから取得する
        public kyous: Array<Kyou> = []
        public mis: Array<Mi> = []
        public tags: Array<Tag> = []
        public texts: Array<Text> = []
        public timeiss: Array<TimeIs> = []

        protected constructor() {
                super()
        }

        async login(_req: LoginRequest): Promise<LoginResponse> {
                throw new Error("not implements")
        }

        async logout(_req: LogoutRequest): Promise<LogoutResponse> {
                throw new Error("not implements")
        }

        async reset_password(_req: ResetPasswordRequest): Promise<ResetPasswordResponse> {
                throw new Error("not implements")
        }

        async set_new_password(_req: SetNewPasswordRequest): Promise<SetNewPasswordResponse> {
                throw new Error("not implements")
        }

        async add_tag(_req: AddTagRequest): Promise<AddTagResponse> {
                throw new Error("not implements")
        }

        async add_text(_req: AddTextRequest): Promise<AddTextResponse> {
                throw new Error("not implements")
        }

        async add_kmemo(_req: AddKmemoRequest): Promise<AddKmemoResponse> {
                throw new Error("not implements")
        }

        async add_urlog(_req: AddURLogRequest): Promise<AddURLogResponse> {
                throw new Error("not implements")
        }

        async add_nlog(_req: AddNlogRequest): Promise<AddNlogResponse> {
                throw new Error("not implements")
        }

        async add_timeis(_req: AddTimeisRequest): Promise<AddTimeisResponse> {
                throw new Error("not implements")
        }

        async add_mi(_req: AddMiRequest): Promise<AddMiResponse> {
                throw new Error("not implements")
        }

        async add_lantana(_req: AddLantanaRequest): Promise<AddLantanaResponse> {
                throw new Error("not implements")
        }

        async add_rekyou(_req: AddReKyouRequest): Promise<AddReKyouResponse> {
                throw new Error("not implements")
        }

        async update_tag(_req: UpdateTagRequest): Promise<UpdateTagResponse> {
                throw new Error("not implements")
        }

        async update_text(_req: UpdateTextRequest): Promise<UpdateTextResponse> {
                throw new Error("not implements")
        }

        async update_kmemo(_req: UpdateKmemoRequest): Promise<UpdateKmemoResponse> {
                throw new Error("not implements")
        }

        async update_urlog(_req: UpdateURLogRequest): Promise<UpdateURLogResponse> {
                throw new Error("not implements")
        }

        async update_nlog(_req: UpdateNlogRequest): Promise<UpdateNlogResponse> {
                throw new Error("not implements")
        }

        async update_timeis(_req: UpdateTimeisRequest): Promise<UpdateTimeisResponse> {
                throw new Error("not implements")
        }

        async update_mi(_req: UpdateMiRequest): Promise<UpdateMiResponse> {
                throw new Error("not implements")
        }

        async update_lantana(_req: UpdateLantanaRequest): Promise<UpdateLantanaResponse> {
                throw new Error("not implements")
        }

        async update_idf_kyou(_req: UpdateIDFKyouRequest): Promise<UpdateIDFKyouResponse> {
                throw new Error("not implements")
        }

        async update_rekyou(_req: UpdateReKyouRequest): Promise<UpdateReKyouResponse> {
                throw new Error("not implements")
        }

        async get_kyous(_req: GetKyousRequest): Promise<GetKyousResponse> {
                throw new Error("not implements")
        }

        async get_kyou(req: GetKyouRequest): Promise<GetKyouResponse> {
                const res = new GetKyouResponse()
                for (let i = 0; i < this.kyous.length; i++) {
                        const kyou = this.kyous[i]
                        if (req.id == kyou.id) {
                                res.kyou_histories.push(kyou)
                        }
                }
                return res
        }

        async get_kmemo(_req: GetKmemoRequest): Promise<GetKmemoResponse> {
                throw new Error("not implements")
        }

        async get_urlog(_req: GetURLogRequest): Promise<GetURLogResponse> {
                throw new Error("not implements")
        }

        async get_nlog(_req: GetNlogRequest): Promise<GetNlogResponse> {
                throw new Error("not implements")
        }

        async get_timeis(req: GetTimeisRequest): Promise<GetTimeisResponse> {
                const res = new GetTimeisResponse()
                for (let i = 0; i < this.timeiss.length; i++) {
                        const timeis = this.timeiss[i]
                        if (req.id == timeis.id) {
                                res.timeis_histories.push(timeis)
                        }
                }
                return res
        }

        async get_mi(req: GetMiRequest): Promise<GetMiResponse> {
                const res = new GetMiResponse()
                for (let i = 0; i < this.mis.length; i++) {
                        const mi = this.mis[i]
                        if (req.id == mi.id) {
                                res.mi_histories.push(mi)
                        }
                }
                return res
        }

        async get_lantana(_req: GetLantanaRequest): Promise<GetLantanaResponse> {
                throw new Error("not implements")
        }

        async get_rekyou(_req: GetReKyouRequest): Promise<GetReKyouResponse> {
                throw new Error("not implements")
        }

        async get_git_commit_log(_req: GetGitCommitLogRequest): Promise<GetGitCommitLogResponse> {
                throw new Error("not implements")
        }

        async get_idf_kyou(_req: GetIDFKyouRequest): Promise<GetIDFKyouResponse> {
                throw new Error("not implements")
        }

        async get_mi_board_list(_req: GetMiBoardRequest): Promise<GetMiBoardResponse> {
                throw new Error("not implements")
        }

        async get_plaing_timeis(_req: GetPlaingTimeisRequest): Promise<GetPlaingTimeisResponse> {
                // not implements
                const res = new GetPlaingTimeisResponse()
                return res
        }

        async get_all_tag_names(_req: GetAllTagNamesRequest): Promise<GetAllTagNamesResponse> {
                throw new Error("not implements")
        }

        async get_all_rep_names(_req: GetAllRepNamesRequest): Promise<GetAllRepNamesResponse> {
                throw new Error("not implements")
        }

        async get_tags_by_target_id(req: GetTagsByTargetIDRequest): Promise<GetTagsByTargetIDResponse> {
                const res = new GetTagsByTargetIDResponse()
                for (let i = 0; i < this.tags.length; i++) {
                        const tag = this.tags[i]
                        if (req.target_id == tag.target_id) {
                                res.tags.push(tag)
                        }
                }
                return res
        }

        async get_tag_histories_by_tag_id(req: GetTagHistoryByTagIDRequest): Promise<GetTagHistoryByTagIDResponse> {
                const res = new GetTagHistoryByTagIDResponse()
                for (let i = 0; i < this.tags.length; i++) {
                        const tag = this.tags[i]
                        if (req.id == tag.id) {
                                res.tag_histories.push(tag)
                        }
                }
                return res
        }

        async get_texts_by_target_id(req: GetTextsByTargetIDRequest): Promise<GetTextsByTargetIDResponse> {
                const res = new GetTextsByTargetIDResponse()
                for (let i = 0; i < this.texts.length; i++) {
                        const text = this.texts[i]
                        if (req.target_id == text.target_id) {
                                res.texts.push(text)
                        }
                }
                return res
        }

        async get_text_history_by_text_id(req: GetTextHistoryByTextIDRequest): Promise<GetTextHistoryByTextIDResponse> {
                const res = new GetTextHistoryByTextIDResponse()
                for (let i = 0; i < this.texts.length; i++) {
                        const text = this.texts[i]
                        if (req.id == text.id) {
                                res.text_histories.push(text)
                        }
                }
                return res
        }

        async get_application_config(_req: GetApplicationConfigRequest): Promise<GetApplicationConfigResponse> {
                throw new Error("not implements")
        }

        async get_server_configs(_req: GetServerConfigsRequest): Promise<GetServerConfigsResponse> {
                throw new Error("not implements")
        }

        async upload_files(_req: UploadFilesRequest): Promise<UploadFilesResponse> {
                throw new Error("not implements")
        }

        async upload_gpslog_files(_req: UploadGPSLogFilesRequest): Promise<UploadGPSLogFilesResponse> {
                throw new Error("not implements")
        }

        async update_application_config(_req: UpdateApplicationConfigRequest): Promise<UpdateApplicationConfigResponse> {
                throw new Error("not implements")
        }

        async update_tag_struct(_req: UpdateTagStructRequest): Promise<UpdateTagStructResponse> {
                throw new Error("not implements")
        }

        async update_rep_struct(_req: UpdateRepStructRequest): Promise<UpdateRepStructResponse> {
                throw new Error("not implements")
        }

        async update_device_struct(_req: UpdateDeviceStructRequest): Promise<UpdateDeviceStructResponse> {
                throw new Error("not implements")
        }

        async update_rep_type_struct(_req: UpdateRepTypeStructRequest): Promise<UpdateRepTypeStructResponse> {
                throw new Error("not implements")
        }

        async update_kftl_template(_req: UpdateKFTLTemplateRequest): Promise<UpdateKFTLTemplateResponse> {
                throw new Error("not implements")
        }

        async update_account_status(_req: UpdateAccountStatusRequest): Promise<UpdateAccountStatusResponse> {
                throw new Error("not implements")
        }

        async update_user_reps(_req: UpdateUserRepsRequest): Promise<UpdateUserRepsResponse> {
                throw new Error("not implements")
        }

        async update_server_config(_req: UpdateServerConfigsRequest): Promise<UpdateServerConfigsResponse> {
                throw new Error("not implements")
        }

        async add_account(_req: AddAccountRequest): Promise<AddAccountResponse> {
                throw new Error("not implements")
        }

        async generate_tls_file(_req: GenerateTLSFileRequest): Promise<GenerateTLSFileResponse> {
                throw new Error("not implements")
        }

        async get_gps_log(_req: GetGPSLogRequest): Promise<GetGPSLogResponse> {
                throw new Error("not implements")
        }

        async get_kftl_templates(_req: GetKFTLTemplatesRequest): Promise<GetKFTLTemplatesResponse> {
                throw new Error("not implements")
        }

        async get_gkill_info(_req: GetGkillInfoRequest): Promise<GetGkillInfoResponse> {
                const res = new GetGkillInfoResponse()
                res.device = "readonly"
                res.user_id = "guest"
                res.user_is_admin = false
                return res
        }

        async get_share_mi_task_list_infos(_req: GetShareMiTaskListInfosRequest): Promise<GetShareMiTaskListInfosResponse> {
                throw new Error("not implements")
        }

        async add_share_mi_task_list_info(_req: AddShareMiTaskListInfoRequest): Promise<AddShareMiTaskListInfoResponse> {
                throw new Error("not implements")
        }

        async update_share_mi_task_list_info(_req: UpdateShareMiTaskListInfoRequest): Promise<UpdateShareMiTaskListInfoResponse> {
                throw new Error("not implements")
        }

        async delete_share_mi_task_list_infos(_req: DeleteShareMiTaskListInfosRequest): Promise<DeleteShareMiTaskListInfosResponse> {
                throw new Error("not implements")
        }

        async get_mi_shared_tasks(req: GetSharedMiTasksRequest): Promise<GetSharedMiTasksResponse> {
                return super.get_mi_shared_tasks(req)
        }

        async get_repositories(_req: GetRepositoriesRequest): Promise<GetRepositoriesResponse> {
                throw new Error("not implements")
        }

        get_session_id(): string {
                return ""
        }

        set_session_id(_session_id: string): void {
                throw new Error("not implements")
        }

        get_google_map_api_key(): string {
                throw new Error("not implements")
        }

        set_google_map_api_key(_google_map_api_key: string): void {
                throw new Error("not implements")
        }

        generate_uuid(): string {
                return super.generate_uuid()
        }

        // 認証が通っていなかったらログイン画面に遷移する
        check_auth(_res: GkillAPIResponse): void {
                throw new Error("not implements")
        }

        set_saved_application_config(_application_config: ApplicationConfig): void {
                throw new Error("not implements")
        }
        get_saved_application_config(): ApplicationConfig | null {
                return super.get_saved_application_config()
        }

        set_saved_rykv_find_kyou_querys(_querys: Array<FindKyouQuery>): void {
                throw new Error("not implements")
        }
        get_saved_rykv_find_kyou_querys(): Array<FindKyouQuery> {
                throw new Error("not implements")
        }

        set_saved_mi_find_kyou_querys(_querys: Array<FindKyouQuery>): void {
                throw new Error("not implements")
        }
        get_saved_mi_find_kyou_querys(): Array<FindKyouQuery> {
                throw new Error("not implements")
        }

        set_saved_rykv_scroll_indexs(_indexs: Array<number>): void {
                throw new Error("not implements")
        }
        get_saved_rykv_scroll_indexs(): Array<number> {
                throw new Error("not implements")
        }

        set_saved_mi_scroll_indexs(_indexs: Array<number>): void {
                throw new Error("not implements")
        }
        get_saved_mi_scroll_indexs(): Array<number> {
                throw new Error("not implements")
        }
}

