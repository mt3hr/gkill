'use strict';

import type { AddAccountRequest } from "./req_res/add-account-request";
import type { AddAccountResponse } from "./req_res/add-account-response";
import type { AddKmemoRequest } from "./req_res/add-kmemo-request";
import type { AddKmemoResponse } from "./req_res/add-kmemo-response";
import type { AddKyouInfoRequest } from "./req_res/add-kyou-info-request";
import type { AddKyouInfoResponse } from "./req_res/add-kyou-info-response";
import type { AddLantanaRequest } from "./req_res/add-lantana-request";
import type { AddLantanaResponse } from "./req_res/add-lantana-response";
import type { AddMiRequest } from "./req_res/add-mi-request";
import type { AddMiResponse } from "./req_res/add-mi-response";
import type { AddNlogRequest } from "./req_res/add-nlog-request";
import type { AddNlogResponse } from "./req_res/add-nlog-response";
import type { AddReKyouRequest } from "./req_res/add-re-kyou-request";
import type { AddReKyouResponse } from "./req_res/add-re-kyou-response";
import type { AddShareMiTaskListInfoRequest } from "./req_res/add-share-mi-task-list-info-request";
import type { AddShareMiTaskListInfoResponse } from "./req_res/add-share-mi-task-list-info-response";
import type { AddTagRequest } from "./req_res/add-tag-request";
import type { AddTagResponse } from "./req_res/add-tag-response";
import type { AddTextRequest } from "./req_res/add-text-request";
import type { AddTextResponse } from "./req_res/add-text-response";
import type { AddTimeisRequest } from "./req_res/add-timeis-request";
import type { AddTimeisResponse } from "./req_res/add-timeis-response";
import type { AddURLogRequest } from "./req_res/add-ur-log-request";
import type { AddURLogResponse } from "./req_res/add-ur-log-response";
import type { DeleteShareMiTaskListInfosRequest } from "./req_res/delete-share-mi-task-list-infos-request";
import type { DeleteShareMiTaskListInfosResponse } from "./req_res/delete-share-mi-task-list-infos-response";
import type { GenerateTLSFileRequest } from "./req_res/generate-tls-file-request";
import type { GenerateTLSFileResponse } from "./req_res/generate-tls-file-response";
import type { GetAllTagNamesRequest } from "./req_res/get-all-tag-names-request";
import type { GetAllTagNamesResponse } from "./req_res/get-all-tag-names-response";
import type { GetApplicationConfigRequest } from "./req_res/get-application-config-request";
import type { GetApplicationConfigResponse } from "./req_res/get-application-config-response";
import type { GetGitCommitLogRequest } from "./req_res/get-git-commit-log-request";
import type { GetGitCommitLogResponse } from "./req_res/get-git-commit-log-response";
import type { GetGitCommitLogsRequest } from "./req_res/get-git-commit-logs-request";
import type { GetGitCommitLogsResponse } from "./req_res/get-git-commit-logs-response";
import type { GetGkillInfoRequest } from "./req_res/get-gkill-info-request";
import type { GetGkillInfoResponse } from "./req_res/get-gkill-info-response";
import type { GetGPSLogRequest } from "./req_res/get-gps-log-request";
import type { GetGPSLogResponse } from "./req_res/get-gps-log-response";
import type { GetKFTLTemplatesRequest } from "./req_res/get-kftl-templates-request";
import type { GetKFTLTemplatesResponse } from "./req_res/get-kftl-templates-response";
import type { GetKmemoRequest } from "./req_res/get-kmemo-request";
import type { GetKmemoResponse } from "./req_res/get-kmemo-response";
import type { GetKmemosRequest } from "./req_res/get-kmemos-request";
import type { GetKmemosResponse } from "./req_res/get-kmemos-response";
import type { GetKyouRequest } from "./req_res/get-kyou-request";
import type { GetKyouResponse } from "./req_res/get-kyou-response";
import type { GetKyousRequest } from "./req_res/get-kyous-request";
import type { GetKyousResponse } from "./req_res/get-kyous-response";
import type { GetLantanaRequest } from "./req_res/get-lantana-request";
import type { GetLantanaResponse } from "./req_res/get-lantana-response";
import type { GetLantanasRequest } from "./req_res/get-lantanas-request";
import type { GetLantanasResponse } from "./req_res/get-lantanas-response";
import type { GetMiBoardRequest } from "./req_res/get-mi-board-request";
import type { GetMiBoardResponse } from "./req_res/get-mi-board-response";
import type { GetMiRequest } from "./req_res/get-mi-request";
import type { GetMiResponse } from "./req_res/get-mi-response";
import type { GetMisRequest } from "./req_res/get-mis-request";
import type { GetMisResponse } from "./req_res/get-mis-response";
import type { GetNlogRequest } from "./req_res/get-nlog-request";
import type { GetNlogResponse } from "./req_res/get-nlog-response";
import type { GetNlogsRequest } from "./req_res/get-nlogs-request";
import type { GetNlogsResponse } from "./req_res/get-nlogs-response";
import type { GetPlaingTimeisRequest } from "./req_res/get-plaing-timeis-request";
import type { GetPlaingTimeisResponse } from "./req_res/get-plaing-timeis-response";
import type { GetReKyouRequest } from "./req_res/get-re-kyou-request";
import type { GetReKyouResponse } from "./req_res/get-re-kyou-response";
import type { GetReKyousRequest } from "./req_res/get-re-kyous-request";
import type { GetReKyousResponse } from "./req_res/get-re-kyous-response";
import type { GetServerConfigRequest } from "./req_res/get-server-config-request";
import type { GetServerConfigResponse } from "./req_res/get-server-config-response";
import type { GetShareMiTaskListInfosRequest } from "./req_res/get-share-mi-task-list-infos-request";
import type { GetShareMiTaskListInfosResponse } from "./req_res/get-share-mi-task-list-infos-response";
import type { GetSharedMiTasksRequest } from "./req_res/get-shared-mi-tasks-request";
import type { GetSharedMiTasksResponse } from "./req_res/get-shared-mi-tasks-response";
import type { GetTagHistoryByTagIDRequest } from "./req_res/get-tag-history-by-tag-id-request";
import type { GetTagHistoryByTagIDResponse } from "./req_res/get-tag-history-by-tag-id-response";
import type { GetTagsByTargetIDRequest } from "./req_res/get-tags-by-target-id-request";
import type { GetTagsByTargetIDResponse } from "./req_res/get-tags-by-target-id-response";
import type { GetTextHistoryByTagIDRequest } from "./req_res/get-text-history-by-tag-id-request";
import type { GetTextHistoryByTagIDResponse } from "./req_res/get-text-history-by-tag-id-response";
import type { GetTextsByTargetIDRequest } from "./req_res/get-texts-by-target-id-request";
import type { GetTextsByTargetIDResponse } from "./req_res/get-texts-by-target-id-response";
import type { GetTimeisRequest } from "./req_res/get-timeis-request";
import type { GetTimeisResponse } from "./req_res/get-timeis-response";
import type { GetTimeissRequest } from "./req_res/get-timeiss-request";
import type { GetTimeissResponse } from "./req_res/get-timeiss-response";
import type { GetURLogRequest } from "./req_res/get-ur-log-request";
import type { GetURLogResponse } from "./req_res/get-ur-log-response";
import type { GetURLogsRequest } from "./req_res/get-ur-logs-request";
import type { GetURLogsResponse } from "./req_res/get-ur-logs-response";
import type { LoginRequest } from "./req_res/login-request";
import type { LoginResponse } from "./req_res/login-response";
import type { LogoutRequest } from "./req_res/logout-request";
import type { LogoutResponse } from "./req_res/logout-response";
import type { ResetPasswordRequest } from "./req_res/reset-password-request";
import type { ResetPasswordResponse } from "./req_res/reset-password-response";
import type { SetNewPasswordRequest } from "./req_res/set-new-password-request";
import type { SetNewPasswordResponse } from "./req_res/set-new-password-response";
import type { UpdateAccountStatusRequest } from "./req_res/update-account-status-request";
import type { UpdateAccountStatusResponse } from "./req_res/update-account-status-response";
import type { UpdateDeviceStructRequest } from "./req_res/update-device-struct-request";
import type { UpdateDeviceStructResponse } from "./req_res/update-device-struct-response";
import type { UpdateKmemoRequest } from "./req_res/update-kmemo-request";
import type { UpdateKmemoResponse } from "./req_res/update-kmemo-response";
import type { UpdateKyouInfoRequest } from "./req_res/update-kyou-info-request";
import type { UpdateKyouInfoResponse } from "./req_res/update-kyou-info-response";
import type { UpdateLantanaRequest } from "./req_res/update-lantana-request";
import type { UpdateLantanaResponse } from "./req_res/update-lantana-response";
import type { UpdateMiRequest } from "./req_res/update-mi-request";
import type { UpdateMiResponse } from "./req_res/update-mi-response";
import type { UpdateNlogRequest } from "./req_res/update-nlog-request";
import type { UpdateNlogResponse } from "./req_res/update-nlog-response";
import type { UpdateReKyouRequest } from "./req_res/update-re-kyou-request";
import type { UpdateReKyouResponse } from "./req_res/update-re-kyou-response";
import type { UpdateRepStructRequest } from "./req_res/update-rep-struct-request";
import type { UpdateRepStructResponse } from "./req_res/update-rep-struct-response";
import type { UpdateRepTypeStructRequest } from "./req_res/update-rep-type-struct-request";
import type { UpdateRepTypeStructResponse } from "./req_res/update-rep-type-struct-response";
import type { UpdateShareMiTaskListInfoRequest } from "./req_res/update-share-mi-task-list-info-request";
import type { UpdateShareMiTaskListInfoResponse } from "./req_res/update-share-mi-task-list-info-response";
import type { UpdateTagRequest } from "./req_res/update-tag-request";
import type { UpdateTagResponse } from "./req_res/update-tag-response";
import type { UpdateTagStructRequest } from "./req_res/update-tag-struct-request";
import type { UpdateTagStructResponse } from "./req_res/update-tag-struct-response";
import type { UpdateTextRequest } from "./req_res/update-text-request";
import type { UpdateTextResponse } from "./req_res/update-text-response";
import type { UpdateTimeisRequest } from "./req_res/update-timeis-request";
import type { UpdateTimeisResponse } from "./req_res/update-timeis-response";
import type { UpdateURLogRequest } from "./req_res/update-ur-log-request";
import type { UpdateURLogResponse } from "./req_res/update-ur-log-response";
import type { UpdateUserRepsRequest } from "./req_res/update-user-reps-request";
import type { UpdateUserRepsResponse } from "./req_res/update-user-reps-response";
import type { UploadFilesRequest } from "./req_res/upload-files-request";
import type { UploadFilesResponse } from "./req_res/upload-files-response";
import type { UploadGPSLogFilesRequest } from "./req_res/upload-gps-log-files-request";
import type { UploadGPSLogFilesResponse } from "./req_res/upload-gps-log-files-response";


export class GkillAPI {


        login_address: string;

        logout_address: string;

        reset_password_address: string;

        set_new_password_address: string;

        add_tag_address: string;

        add_text_address: string;

        add_kmemo_address: string;

        add_urlog_address: string;

        add_nlog_address: string;

        add_timeis_address: string;

        add_mi_address: string;

        add_lantana_address: string;

        add_kyou_info_address: string;

        add_rekyou_address: string;

        update_tag_address: string;

        update_text_address: string;

        update_kmemo_address: string;

        update_urlog_address: string;

        update_nlog_address: string;

        update_timeis_address: string;

        update_lantana_address: string;

        update_mi_address: string;

        update_kyou_info_address: string;

        update_rekyou_address: string;

        get_kyous_address: string;

        get_kyou_address: string;

        get_kmemos_address: string;

        get_urlogs_address: string;

        get_nlogs_address: string;

        get_timeiss_address: string;

        get_rekyous_address: string;

        get_mis_address: string;

        get_lantanas_address: string;

        get_kmemo_address: string;

        get_urlog_address: string;

        get_nlog_address: string;

        get_timeis_address: string;

        get_mi_address: string;

        get_lantana_address: string;

        get_rekyou_address: string;

        get_git_commit_log_address: string;

        get_git_commit_logs_address: string;

        get_mi_board_list_address: string;

        get_plaing_timeis_address: string;

        get_all_tag_names_address: string;

        get_tags_by_target_id_address: string;

        get_tag_histories_by_tag_id_address: string;

        get_texts_by_target_id_address: string;

        get_text_histories_by_text_id_address: string;

        get_application_config_address: string;

        get_server_config_address: string;

        upload_files_address: string;

        upload_gpslog_files_address: string;

        update_tag_struct_address: string;

        update_rep_struct_address: string;

        update_device_struct_address: string;

        update_rep_type_struct_address: string;

        update_account_status_address: string;

        update_user_reps_address: string;

        add_account_address: string;

        generate_tls_file_address: string;

        get_gps_log_address: string;

        get_kftl_template_address: string;

        get_gkill_info_address: string;

        get_share_mi_task_list_infos_address: string;

        add_share_mi_task_list_info_address: string;

        update_share_mi_task_list_info_address: string;

        delete_share_mi_task_list_infos_address: string;

        get_mi_shared_tasks_address: string;

        login_method: string;

        logout_method: string;

        reset_password_method: string;

        set_new_password_method: string;

        add_tag_method: string;

        add_text_method: string;

        add_kmemo_method: string;

        add_urlog_method: string;

        add_nlog_method: string;

        add_timeis_method: string;

        add_mi_method: string;

        add_lantana_method: string;

        add_kyou_info_method: string;

        add_rekyou_method: string;

        update_tag_method: string;

        update_text_method: string;

        update_kmemo_method: string;

        update_nlog_method: string;

        update_urlog_method: string;

        update_timeis_method: string;

        update_mi_method: string;

        update_lantana_method: string;

        update_kyou_info_method: string;

        update_rekyou_method: string;

        get_kyous_method: string;

        get_kyou_method: string;

        get_rekyou_method: string;

        get_rekyous_method: string;

        get_git_commit_logs_method: string;

        get_mi_board_list_method: string;

        get_plaing_timeis_method: string;

        get_all_tag_names_method: string;

        get_tags_by_target_id_method: string;

        get_tag_histories_by_tag_id_method: string;

        get_texts_by_target_id_method: string;

        get_text_histories_by_tag_id_method: string;

        get_application_config_method: string;

        get_server_config_method: string;

        upload_files_method: string;

        upload_gpslog_files_method: string;

        update_tag_struct_method: string;

        update_rep_struct_method: string;

        update_device_struct_method: string;

        update_rep_type_struct_method: string;

        update_account_status_method: string;

        update_user_reps_method: string;

        add_account_method: string;

        generate_tls_file_method: string;

        get_gps_log_method: string;

        get_kftl_template_method: string;

        get_gkill_info_method: string;

        get_share_mi_task_list_infos_method: string;

        add_share_mi_task_list_info_method: string;

        update_share_mi_task_list_info_method: string;

        delete_share_mi_task_list_infos_method: string;

        get_mi_shared_tasks_method: string;

        constructor() {
                this.login_address = "/api/login";
                this.logout_address = "/api/logout";
                this.reset_password_address = "/api/reset_password";
                this.set_new_password_address = "/api/set_new_password";
                this.add_tag_address = "/api/add_tag";
                this.add_text_address = "/api/add_text";
                this.add_kmemo_address = "/api/add_kmemo";
                this.add_urlog_address = "/api/add_urlog";
                this.add_nlog_address = "/api/add_nlog";
                this.add_timeis_address = "/api/add_timeis";
                this.add_mi_address = "/api/add_mi";
                this.add_lantana_address = "/api/add_lantana";
                this.add_kyou_info_address = "/api/add_kyou_info";
                this.add_rekyou_address = "/api/add_rekyou";
                this.update_tag_address = "/api/update_tag";
                this.update_text_address = "/api/update_text";
                this.update_kmemo_address = "/api/update_kmemo";
                this.update_urlog_address = "/api/update_urlog";
                this.update_nlog_address = "/api/update_nlog";
                this.update_timeis_address = "/api/update_timeis";
                this.update_lantana_address = "/api/update_lantana";
                this.update_mi_address = "/api/update_mi";
                this.update_kyou_info_address = "/api/update_kyou_info";
                this.update_rekyou_address = "/api/update_rekyou";
                this.get_kyous_address = "/api/get_kyous";
                this.get_kyou_address = "/api/get_kyou";
                this.get_kmemos_address = "/api/get_kmemos";
                this.get_urlogs_address = "/api/get_urlogs";
                this.get_nlogs_address = "/api/get_nlogs";
                this.get_timeiss_address = "/api/get_timeiss";
                this.get_rekyous_address = "/api/get_rekyous";
                this.get_mis_address = "/api/get_mis";
                this.get_lantanas_address = "/api/get_lantanas";
                this.get_kmemo_address = "/api/get_kmemo";
                this.get_urlog_address = "/api/get_urlog";
                this.get_nlog_address = "/api/get_nlog";
                this.get_timeis_address = "/api/get_timeis";
                this.get_mi_address = "/api/get_mi";
                this.get_lantana_address = "/api/get_lantana";
                this.get_rekyou_address = "/api/get_rekyou";
                this.get_git_commit_log_address = "/api/get_git_commit_log";
                this.get_git_commit_logs_address = "/api/get_git_commit_logs";
                this.get_mi_board_list_address = "/api/get_mi_board_list";
                this.get_plaing_timeis_address = "/api/get_plaing_timeis";
                this.get_all_tag_names_address = "/api/get_all_tag_names";
                this.get_tags_by_target_id_address = "/api/get_tags_by_id";
                this.get_tag_histories_by_tag_id_address = "/api/get_tag_histories_by_tag_id";
                this.get_texts_by_target_id_address = "/api/get_texts_by_id";
                this.get_text_histories_by_text_id_address = "/api/get_text_histories_by_text_id";
                this.get_application_config_address = "/api/get_application_config";
                this.get_server_config_address = "/api/get_server_config";
                this.upload_files_address = "/api/upload_files";
                this.upload_gpslog_files_address = "/api/upload_gpslog_files";
                this.update_tag_struct_address = "/api/update_tag_struct";
                this.update_rep_struct_address = "/api/update_rep_struct";
                this.update_device_struct_address = "/api/update_device_struct";
                this.update_rep_type_struct_address = "/api/update_rep_struct";
                this.update_account_status_address = "/api/update_account_status";
                this.update_user_reps_address = "/api/update_user_reps";
                this.add_account_address = "/api/add_user";
                this.generate_tls_file_address = "/api/generate_tls_file";
                this.get_gps_log_address = "/api/get_gps_log";
                this.get_kftl_template_address = "/api/get_kftl_template";
                this.get_gkill_info_address = "/api/get_gkill_info";
                this.get_share_mi_task_list_infos_address = "/api/get_share_mi_task_list_infos";
                this.add_share_mi_task_list_info_address = "/api/add_share_mi_task_list_info";
                this.update_share_mi_task_list_info_address = "/api/update_share_mi_task_list_info";
                this.delete_share_mi_task_list_infos_address = "/api/delete_share_mi_task_list_infos";
                this.get_mi_shared_tasks_address = "/api/get_mi_shared_tasks";
                this.login_method = "POST";
                this.logout_method = "POST";
                this.reset_password_method = "POST";
                this.set_new_password_method = "POST";
                this.add_tag_method = "POST";
                this.add_text_method = "POST";
                this.add_kmemo_method = "POST";
                this.add_urlog_method = "POST";
                this.add_nlog_method = "POST";
                this.add_timeis_method = "POST";
                this.add_mi_method = "POST";
                this.add_lantana_method = "POST";
                this.add_kyou_info_method = "POST";
                this.add_rekyou_method = "POST";
                this.update_tag_method = "POST";
                this.update_text_method = "POST";
                this.update_kmemo_method = "POST";
                this.update_nlog_method = "POST";
                this.update_urlog_method = "POST";
                this.update_timeis_method = "POST";
                this.update_mi_method = "POST";
                this.update_lantana_method = "POST";
                this.update_kyou_info_method = "POST";
                this.update_rekyou_method = "POST";
                this.get_kyous_method = "POST";
                this.get_kyou_method = "POST";
                this.get_rekyou_method = "POST";
                this.get_rekyous_method = "POST";
                this.get_git_commit_logs_method = "POST";
                this.get_mi_board_list_method = "POST";
                this.get_plaing_timeis_method = "POST";
                this.get_all_tag_names_method = "POST";
                this.get_tags_by_target_id_method = "POST";
                this.get_tag_histories_by_tag_id_method = "POST";
                this.get_texts_by_target_id_method = "POST";
                this.get_text_histories_by_tag_id_method = "POST";
                this.get_application_config_method = "POST";
                this.get_server_config_method = "POST";
                this.upload_files_method = "POST";
                this.upload_gpslog_files_method = "POST";
                this.update_tag_struct_method = "POST";
                this.update_rep_struct_method = "POST";
                this.update_device_struct_method = "POST";
                this.update_rep_type_struct_method = "POST";
                this.update_account_status_method = "POST";
                this.update_user_reps_method = "POST";
                this.add_account_method = "POST";
                this.generate_tls_file_method = "POST";
                this.get_gps_log_method = "POST";
                this.get_kftl_template_method = "POST";
                this.get_gkill_info_method = "POST";
                this.get_share_mi_task_list_infos_method = "POST";
                this.add_share_mi_task_list_info_method = "POST";
                this.update_share_mi_task_list_info_method = "POST";
                this.delete_share_mi_task_list_infos_method = "POST";
                this.get_mi_shared_tasks_method = "POST";

        }

        async login(req: LoginRequest): Promise<LoginResponse> {
                throw new Error('Not implemented');
        }

        async logout(req: LogoutRequest): Promise<LogoutResponse> {
                throw new Error('Not implemented');
        }

        async reset_password(req: ResetPasswordRequest): Promise<ResetPasswordResponse> {
                throw new Error('Not implemented');
        }

        async set_new_password(req: SetNewPasswordRequest): Promise<SetNewPasswordResponse> {
                throw new Error('Not implemented');
        }

        async add_tag(req: AddTagRequest): Promise<AddTagResponse> {
                throw new Error('Not implemented');
        }

        async add_text(req: AddTextRequest): Promise<AddTextResponse> {
                throw new Error('Not implemented');
        }

        async add_kmemo(req: AddKmemoRequest): Promise<AddKmemoResponse> {
                throw new Error('Not implemented');
        }

        async add_urlog(req: AddURLogRequest): Promise<AddURLogResponse> {
                throw new Error('Not implemented');
        }

        async add_nlog(req: AddNlogRequest): Promise<AddNlogResponse> {
                throw new Error('Not implemented');
        }

        async add_timeis(req: AddTimeisRequest): Promise<AddTimeisResponse> {
                print()
                throw new Error('Not implemented');
        }

        async add_mi(req: AddMiRequest): Promise<AddMiResponse> {
                throw new Error('Not implemented');
        }

        async add_lantana(req: AddLantanaRequest): Promise<AddLantanaResponse> {
                throw new Error('Not implemented');
        }

        async add_kyou_info(req: AddKyouInfoRequest): Promise<AddKyouInfoResponse> {
                throw new Error('Not implemented');
        }

        async add_rekyou(req: AddReKyouRequest): Promise<AddReKyouResponse> {
                throw new Error('Not implemented');
        }

        async update_tag(req: UpdateTagRequest): Promise<UpdateTagResponse> {
                throw new Error('Not implemented');
        }

        async update_text(req: UpdateTextRequest): Promise<UpdateTextResponse> {
                throw new Error('Not implemented');
        }

        async update_kmemo(req: UpdateKmemoRequest): Promise<UpdateKmemoResponse> {
                throw new Error('Not implemented');
        }

        async update_urlog(req: UpdateURLogRequest): Promise<UpdateURLogResponse> {
                throw new Error('Not implemented');
        }

        async update_nlog(req: UpdateNlogRequest): Promise<UpdateNlogResponse> {
                throw new Error('Not implemented');
        }

        async update_timeis(req: UpdateTimeisRequest): Promise<UpdateTimeisResponse> {
                throw new Error('Not implemented');
        }

        async update_mi(req: UpdateMiRequest): Promise<UpdateMiResponse> {
                throw new Error('Not implemented');
        }

        async update_lantana(req: UpdateLantanaRequest): Promise<UpdateLantanaResponse> {
                throw new Error('Not implemented');
        }

        async update_kyou_info(req: UpdateKyouInfoRequest): Promise<UpdateKyouInfoResponse> {
                throw new Error('Not implemented');
        }

        async update_rekyou(req: UpdateReKyouRequest): Promise<UpdateReKyouResponse> {
                throw new Error('Not implemented');
        }

        async get_kyous(req: GetKyousRequest): Promise<GetKyousResponse> {
                throw new Error('Not implemented');
        }

        async get_kyou(req: GetKyouRequest): Promise<GetKyouResponse> {
                throw new Error('Not implemented');
        }

        async get_kmemo(req: GetKmemoRequest): Promise<GetKmemoResponse> {
                throw new Error('Not implemented');
        }

        async get_urlog(req: GetURLogRequest): Promise<GetURLogResponse> {
                throw new Error('Not implemented');
        }

        async get_nlog(req: GetNlogRequest): Promise<GetNlogResponse> {
                throw new Error('Not implemented');
        }

        async get_timeis(req: GetTimeisRequest): Promise<GetTimeisResponse> {
                throw new Error('Not implemented');
        }

        async get_mi(req: GetMiRequest): Promise<GetMiResponse> {
                throw new Error('Not implemented');
        }

        async get_lantana(req: GetLantanaRequest): Promise<GetLantanaResponse> {
                throw new Error('Not implemented');
        }

        async get_rekyou(req: GetReKyouRequest): Promise<GetReKyouResponse> {
                throw new Error('Not implemented');
        }

        async get_git_commit_log(req: GetGitCommitLogRequest): Promise<GetGitCommitLogResponse> {
                throw new Error('Not implemented');
        }

        async get_kmemos(req: GetKmemosRequest): Promise<GetKmemosResponse> {
                throw new Error('Not implemented');
        }

        async get_urlogs(req: GetURLogsRequest): Promise<GetURLogsResponse> {
                throw new Error('Not implemented');
        }

        async get_nlogs(req: GetNlogsRequest): Promise<GetNlogsResponse> {
                throw new Error('Not implemented');
        }

        async get_timeiss(req: GetTimeissRequest): Promise<GetTimeissResponse> {
                throw new Error('Not implemented');
        }

        async get_rekyous(req: GetReKyousRequest): Promise<GetReKyousResponse> {
                throw new Error('Not implemented');
        }

        async get_mis(req: GetMisRequest): Promise<GetMisResponse> {
                throw new Error('Not implemented');
        }

        async get_lantanas(req: GetLantanasRequest): Promise<GetLantanasResponse> {
                throw new Error('Not implemented');
        }

        async get_git_commit_logs(req: GetGitCommitLogsRequest): Promise<GetGitCommitLogsResponse> {
                throw new Error('Not implemented');
        }

        async get_mi_board_list(req: GetMiBoardRequest): Promise<GetMiBoardResponse> {
                throw new Error('Not implemented');
        }

        async get_plaing_timeis(req: GetPlaingTimeisRequest): Promise<GetPlaingTimeisResponse> {
                throw new Error('Not implemented');
        }

        async get_all_tag_names(req: GetAllTagNamesRequest): Promise<GetAllTagNamesResponse> {
                throw new Error('Not implemented');
        }

        async get_tags_by_target_id(req: GetTagsByTargetIDRequest): Promise<GetTagsByTargetIDResponse> {
                throw new Error('Not implemented');
        }

        async get_tag_histories_by_tag_id(req: GetTagHistoryByTagIDRequest): Promise<GetTagHistoryByTagIDResponse> {
                throw new Error('Not implemented');
        }

        async get_texts_by_target_id(req: GetTextsByTargetIDRequest): Promise<GetTextsByTargetIDResponse> {
                throw new Error('Not implemented');
        }

        async get_text_history_by_tag_id(req: GetTextHistoryByTagIDRequest): Promise<GetTextHistoryByTagIDResponse> {
                throw new Error('Not implemented');
        }

        async get_application_config(req: GetApplicationConfigRequest): Promise<GetApplicationConfigResponse> {
                throw new Error('Not implemented');
        }

        async get_server_config(req: GetServerConfigRequest): Promise<GetServerConfigResponse> {
                throw new Error('Not implemented');
        }

        async upload_files(req: UploadFilesRequest): Promise<UploadFilesResponse> {
                throw new Error('Not implemented');
        }

        async upload_gpslog_files(req: UploadGPSLogFilesRequest): Promise<UploadGPSLogFilesResponse> {
                throw new Error('Not implemented');
        }

        async update_tag_struct(req: UpdateTagStructRequest): Promise<UpdateTagStructResponse> {
                throw new Error('Not implemented');
        }

        async update_rep_struct(req: UpdateRepStructRequest): Promise<UpdateRepStructResponse> {
                throw new Error('Not implemented');
        }

        async update_decice_struct(req: UpdateDeviceStructRequest): Promise<UpdateDeviceStructResponse> {
                throw new Error('Not implemented');
        }

        async update_rep_type_struct(req: UpdateRepTypeStructRequest): Promise<UpdateRepTypeStructResponse> {
                throw new Error('Not implemented');
        }

        async update_account_status(req: UpdateAccountStatusRequest): Promise<UpdateAccountStatusResponse> {
                throw new Error('Not implemented');
        }

        async update_user_reps(req: UpdateUserRepsRequest): Promise<UpdateUserRepsResponse> {
                throw new Error('Not implemented');
        }

        async add_account(req: AddAccountRequest): Promise<AddAccountResponse> {
                throw new Error('Not implemented');
        }

        async generate_tls_file(req: GenerateTLSFileRequest): Promise<GenerateTLSFileResponse> {
                throw new Error('Not implemented');
        }

        async get_gps_log(req: GetGPSLogRequest): Promise<GetGPSLogResponse> {
                throw new Error('Not implemented');
        }

        async get_kftl_templates(req: GetKFTLTemplatesRequest): Promise<GetKFTLTemplatesResponse> {
                throw new Error('Not implemented');
        }

        async get_gkill_info(req: GetGkillInfoRequest): Promise<GetGkillInfoResponse> {
                throw new Error('Not implemented');
        }

        async get_share_mi_task_list_infos(req: GetShareMiTaskListInfosRequest): Promise<GetShareMiTaskListInfosResponse> {
                throw new Error('Not implemented');
        }

        async add_share_mi_task_list_info(req: AddShareMiTaskListInfoRequest): Promise<AddShareMiTaskListInfoResponse> {
                throw new Error('Not implemented');
        }

        async update_share_mi_task_list_info(req: UpdateShareMiTaskListInfoRequest): Promise<UpdateShareMiTaskListInfoResponse> {
                throw new Error('Not implemented');
        }

        async delete_share_mi_task_list_infos(req: DeleteShareMiTaskListInfosRequest): Promise<DeleteShareMiTaskListInfosResponse> {
                throw new Error('Not implemented');
        }

        async get_mi_shared_tasks(req: GetSharedMiTasksRequest): Promise<GetSharedMiTasksResponse> {
                throw new Error('Not implemented');
        }


}



