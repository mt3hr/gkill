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
import { GetKyousRequest } from "./req_res/get-kyous-request"
import { GetKyousResponse } from "./req_res/get-kyous-response"
import type { GetLantanaRequest } from "./req_res/get-lantana-request"
import type { GetLantanaResponse } from "./req_res/get-lantana-response"
import type { GetMiBoardRequest } from "./req_res/get-mi-board-request"
import type { GetMiBoardResponse } from "./req_res/get-mi-board-response"
import type { GetMiRequest } from "./req_res/get-mi-request"
import { GetMiResponse } from "./req_res/get-mi-response"
import type { GetNlogRequest } from "./req_res/get-nlog-request"
import type { GetNlogResponse } from "./req_res/get-nlog-response"
import { GetPlaingTimeisRequest } from "./req_res/get-plaing-timeis-request"
import { GetPlaingTimeisResponse } from "./req_res/get-plaing-timeis-response"
import type { GetReKyouRequest } from "./req_res/get-re-kyou-request"
import type { GetReKyouResponse } from "./req_res/get-re-kyou-response"
import type { GetShareMiTaskListInfosRequest } from "./req_res/get-share-mi-task-list-infos-request"
import { GetShareMiTaskListInfosResponse } from "./req_res/get-share-mi-task-list-infos-response"
import type { GetSharedMiTasksRequest } from "./req_res/get-shared-mi-tasks-request"
import { GetSharedMiTasksResponse } from "./req_res/get-shared-mi-tasks-response"
import { GetTagHistoryByTagIDRequest } from "./req_res/get-tag-history-by-tag-id-request"
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
import type { UpdateKFTLTemplateRequest } from "./req_res/update-kftl-template-request"
import type { UpdateKFTLTemplateResponse } from "./req_res/update-kftl-template-response"
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
import { GkillAPI } from "./gkill-api"

export class GkillAPIForSharedMi extends GkillAPI {
        private static gkill_api_for_shared_mi: GkillAPIForSharedMi = new GkillAPIForSharedMi()
        static get_instance(): GkillAPIForSharedMi {
                return this.gkill_api_for_shared_mi
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

        async login(req: LoginRequest): Promise<LoginResponse> {
                throw new Error("not implements")
        }

        async logout(req: LogoutRequest): Promise<LogoutResponse> {
                throw new Error("not implements")
        }

        async reset_password(req: ResetPasswordRequest): Promise<ResetPasswordResponse> {
                throw new Error("not implements")
        }

        async set_new_password(req: SetNewPasswordRequest): Promise<SetNewPasswordResponse> {
                throw new Error("not implements")
        }

        async add_tag(req: AddTagRequest): Promise<AddTagResponse> {
                throw new Error("not implements")
        }

        async add_text(req: AddTextRequest): Promise<AddTextResponse> {
                throw new Error("not implements")
        }

        async add_kmemo(req: AddKmemoRequest): Promise<AddKmemoResponse> {
                throw new Error("not implements")
        }

        async add_urlog(req: AddURLogRequest): Promise<AddURLogResponse> {
                throw new Error("not implements")
        }

        async add_nlog(req: AddNlogRequest): Promise<AddNlogResponse> {
                throw new Error("not implements")
        }

        async add_timeis(req: AddTimeisRequest): Promise<AddTimeisResponse> {
                throw new Error("not implements")
        }

        async add_mi(req: AddMiRequest): Promise<AddMiResponse> {
                throw new Error("not implements")
        }

        async add_lantana(req: AddLantanaRequest): Promise<AddLantanaResponse> {
                throw new Error("not implements")
        }

        async add_rekyou(req: AddReKyouRequest): Promise<AddReKyouResponse> {
                throw new Error("not implements")
        }

        async update_tag(req: UpdateTagRequest): Promise<UpdateTagResponse> {
                throw new Error("not implements")
        }

        async update_text(req: UpdateTextRequest): Promise<UpdateTextResponse> {
                throw new Error("not implements")
        }

        async update_kmemo(req: UpdateKmemoRequest): Promise<UpdateKmemoResponse> {
                throw new Error("not implements")
        }

        async update_urlog(req: UpdateURLogRequest): Promise<UpdateURLogResponse> {
                throw new Error("not implements")
        }

        async update_nlog(req: UpdateNlogRequest): Promise<UpdateNlogResponse> {
                throw new Error("not implements")
        }

        async update_timeis(req: UpdateTimeisRequest): Promise<UpdateTimeisResponse> {
                throw new Error("not implements")
        }

        async update_mi(req: UpdateMiRequest): Promise<UpdateMiResponse> {
                throw new Error("not implements")
        }

        async update_lantana(req: UpdateLantanaRequest): Promise<UpdateLantanaResponse> {
                throw new Error("not implements")
        }

        async update_idf_kyou(req: UpdateIDFKyouRequest): Promise<UpdateIDFKyouResponse> {
                throw new Error("not implements")
        }

        async update_rekyou(req: UpdateReKyouRequest): Promise<UpdateReKyouResponse> {
                throw new Error("not implements")
        }

        async get_kyous(req: GetKyousRequest): Promise<GetKyousResponse> {
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

        async get_kmemo(req: GetKmemoRequest): Promise<GetKmemoResponse> {
                throw new Error("not implements")
        }

        async get_urlog(req: GetURLogRequest): Promise<GetURLogResponse> {
                throw new Error("not implements")
        }

        async get_nlog(req: GetNlogRequest): Promise<GetNlogResponse> {
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

        async get_lantana(req: GetLantanaRequest): Promise<GetLantanaResponse> {
                throw new Error("not implements")
        }

        async get_rekyou(req: GetReKyouRequest): Promise<GetReKyouResponse> {
                throw new Error("not implements")
        }

        async get_git_commit_log(req: GetGitCommitLogRequest): Promise<GetGitCommitLogResponse> {
                throw new Error("not implements")
        }

        async get_idf_kyou(req: GetIDFKyouRequest): Promise<GetIDFKyouResponse> {
                throw new Error("not implements")
        }

        async get_mi_board_list(req: GetMiBoardRequest): Promise<GetMiBoardResponse> {
                throw new Error("not implements")
        }

        async get_plaing_timeis(req: GetPlaingTimeisRequest): Promise<GetPlaingTimeisResponse> {
                // not implements
                const res = new GetPlaingTimeisResponse()
                return res
        }

        async get_all_tag_names(req: GetAllTagNamesRequest): Promise<GetAllTagNamesResponse> {
                throw new Error("not implements")
        }

        async get_all_rep_names(req: GetAllRepNamesRequest): Promise<GetAllRepNamesResponse> {
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

        async get_application_config(req: GetApplicationConfigRequest): Promise<GetApplicationConfigResponse> {
                throw new Error("not implements")
        }

        async get_server_configs(req: GetServerConfigsRequest): Promise<GetServerConfigsResponse> {
                throw new Error("not implements")
        }

        async upload_files(req: UploadFilesRequest): Promise<UploadFilesResponse> {
                throw new Error("not implements")
        }

        async upload_gpslog_files(req: UploadGPSLogFilesRequest): Promise<UploadGPSLogFilesResponse> {
                throw new Error("not implements")
        }

        async update_application_config(req: UpdateApplicationConfigRequest): Promise<UpdateApplicationConfigResponse> {
                throw new Error("not implements")
        }

        async update_tag_struct(req: UpdateTagStructRequest): Promise<UpdateTagStructResponse> {
                throw new Error("not implements")
        }

        async update_rep_struct(req: UpdateRepStructRequest): Promise<UpdateRepStructResponse> {
                throw new Error("not implements")
        }

        async update_device_struct(req: UpdateDeviceStructRequest): Promise<UpdateDeviceStructResponse> {
                throw new Error("not implements")
        }

        async update_rep_type_struct(req: UpdateRepTypeStructRequest): Promise<UpdateRepTypeStructResponse> {
                throw new Error("not implements")
        }

        async update_kftl_template(req: UpdateKFTLTemplateRequest): Promise<UpdateKFTLTemplateResponse> {
                throw new Error("not implements")
        }

        async update_account_status(req: UpdateAccountStatusRequest): Promise<UpdateAccountStatusResponse> {
                throw new Error("not implements")
        }

        async update_user_reps(req: UpdateUserRepsRequest): Promise<UpdateUserRepsResponse> {
                throw new Error("not implements")
        }

        async update_server_config(req: UpdateServerConfigsRequest): Promise<UpdateServerConfigsResponse> {
                throw new Error("not implements")
        }

        async add_account(req: AddAccountRequest): Promise<AddAccountResponse> {
                throw new Error("not implements")
        }

        async generate_tls_file(req: GenerateTLSFileRequest): Promise<GenerateTLSFileResponse> {
                throw new Error("not implements")
        }

        async get_gps_log(req: GetGPSLogRequest): Promise<GetGPSLogResponse> {
                throw new Error("not implements")
        }

        async get_kftl_templates(req: GetKFTLTemplatesRequest): Promise<GetKFTLTemplatesResponse> {
                throw new Error("not implements")
        }

        async get_gkill_info(req: GetGkillInfoRequest): Promise<GetGkillInfoResponse> {
                const res = new GetGkillInfoResponse()
                res.device = "readonly"
                res.user_id = "guest"
                res.user_is_admin = false
                return res
        }

        async get_share_mi_task_list_infos(req: GetShareMiTaskListInfosRequest): Promise<GetShareMiTaskListInfosResponse> {
                throw new Error("not implements")
        }

        async add_share_mi_task_list_info(req: AddShareMiTaskListInfoRequest): Promise<AddShareMiTaskListInfoResponse> {
                throw new Error("not implements")
        }

        async update_share_mi_task_list_info(req: UpdateShareMiTaskListInfoRequest): Promise<UpdateShareMiTaskListInfoResponse> {
                throw new Error("not implements")
        }

        async delete_share_mi_task_list_infos(req: DeleteShareMiTaskListInfosRequest): Promise<DeleteShareMiTaskListInfosResponse> {
                throw new Error("not implements")
        }

        async get_mi_shared_tasks(req: GetSharedMiTasksRequest): Promise<GetSharedMiTasksResponse> {
                return super.get_mi_shared_tasks(req)
        }

        async get_repositories(req: GetRepositoriesRequest): Promise<GetRepositoriesResponse> {
                throw new Error("not implements")
        }

        get_session_id(): string {
                return ""
        }

        set_session_id(session_id: string): void {
                throw new Error("not implements")
        }

        get_google_map_api_key(): string {
                throw new Error("not implements")
        }

        set_google_map_api_key(google_map_api_key: string): void {
                throw new Error("not implements")
        }

        generate_uuid(): string {
                return super.generate_uuid()
        }

        // 認証が通っていなかったらログイン画面に遷移する
        check_auth(res: GkillAPIResponse): void {
                throw new Error("not implements")
        }

        set_saved_application_config(application_config: ApplicationConfig): void {
                throw new Error("not implements")
        }
        get_saved_application_config(): ApplicationConfig | null {
                return super.get_saved_application_config()
        }

        set_saved_rykv_find_kyou_querys(querys: Array<FindKyouQuery>): void {
                throw new Error("not implements")
        }
        get_saved_rykv_find_kyou_querys(): Array<FindKyouQuery> {
                throw new Error("not implements")
        }

        set_saved_mi_find_kyou_querys(querys: Array<FindKyouQuery>): void {
                throw new Error("not implements")
        }
        get_saved_mi_find_kyou_querys(): Array<FindKyouQuery> {
                throw new Error("not implements")
        }

        set_saved_rykv_scroll_indexs(indexs: Array<number>): void {
                throw new Error("not implements")
        }
        get_saved_rykv_scroll_indexs(): Array<number> {
                throw new Error("not implements")
        }

        set_saved_mi_scroll_indexs(indexs: Array<number>): void {
                throw new Error("not implements")
        }
        get_saved_mi_scroll_indexs(): Array<number> {
                throw new Error("not implements")
        }
}


