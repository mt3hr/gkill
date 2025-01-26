<template>
    <div v-show="target_rep_name_for_file !== '' && target_rep_name_for_gps_file !== ''">
        <v-card>
            <v-card-title>
                ファイルアップロード
            </v-card-title>
            <v-tabs v-model="tab">
                <v-tab key="file">ファイル</v-tab>
                <v-tab key="gps_log_file">GPSLogファイル</v-tab>
            </v-tabs>
            <v-window v-model="tab">
                <v-window-item key="file" :eager="true">
                    <v-card>
                        <v-card-title>
                            ファイル名衝突時
                        </v-card-title>
                        <v-radio-group v-model="conflict_behavior_file">
                            <v-radio label="別名保存" :value="'rename'" />
                            <v-radio label="上書き" :value="'override'" />
                        </v-radio-group>
                        <v-select class="select" v-model="target_rep_name_for_file" :items="target_rep_names_for_file"
                            label="アップロード先" />
                        <v-file-input label="ファイル" multiple @change="upload_files" v-model="files" />
                    </v-card>
                </v-window-item>
                <v-window-item key="gps_log_file" :eager="true">
                    <v-card>
                        <v-card-title>
                            ファイル名衝突時
                        </v-card-title>
                        <v-radio-group v-model="conflict_behavior_gps_file">
                            <v-radio label="マージ" :value="'merge'" />
                            <v-radio label="上書き" :value="'override'" />
                        </v-radio-group>
                        <v-select class="select" v-model="target_rep_name_for_gps_file"
                            :items="target_rep_names_for_gps_file" label="アップロード先" />
                        <v-file-input label="GPSLogファイル" multiple @change="upload_gps_log_files" v-model="gps_log_files"
                            accept=".gpx" />
                    </v-card>
                </v-window-item>
            </v-window>
        </v-card>
        <DecideRelatedTimeUploadedFileDialog :app_content_height="app_content_height"
            :app_content_width="app_content_width" :application_config="application_config" :gkill_api="gkill_api"
            :uploaded_kyous="uploaded_kyous" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => reload_kyou(kyou)" ref="decide_related_time_uploaded_file_dialog" />
        <ProgressUploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="progress_upload_file_dialog" />
        <ProgressUploadGPSFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            ref="progress_upload_gps_file_dialog" />
    </div>
</template>
<script setup lang="ts">
import { nextTick, type Ref, ref } from 'vue'
import DecideRelatedTimeUploadedFileDialog from '../dialogs/decide-related-time-uploaded-file-dialog.vue'
import ProgressUploadFileDialog from '../dialogs/progress-upload-file-dialog.vue'
import ProgressUploadGPSFileDialog from '../dialogs/progress-upload-gps-file-dialog.vue'

import type { UploadFileViewEmits } from './upload-file-view-emits'
import type { UploadFileViewProps } from './upload-file-view-props'
import type { Kyou } from '@/classes/datas/kyou'
import { FileUploadConflictBehavior } from '@/classes/api/req_res/file-upload-conflict-behavior'
import { UploadGPSLogFilesRequest } from '@/classes/api/req_res/upload-gps-log-files-request'
import { FileData } from '@/classes/api/file-data'
import { UploadFilesRequest } from '@/classes/api/req_res/upload-files-request'
import { GetRepositoriesRequest } from '@/classes/api/req_res/get-repositories-request'
import type { Repository } from '@/classes/datas/config/repository'

const progress_upload_file_dialog = ref<InstanceType<typeof ProgressUploadFileDialog> | null>(null);
const progress_upload_gps_file_dialog = ref<InstanceType<typeof ProgressUploadGPSFileDialog> | null>(null);
const decide_related_time_uploaded_file_dialog = ref<InstanceType<typeof DecideRelatedTimeUploadedFileDialog> | null>(null);

const tab = ref(2)

const props = defineProps<UploadFileViewProps>()
const emits = defineEmits<UploadFileViewEmits>()
const conflict_behavior_file: Ref<FileUploadConflictBehavior> = ref(FileUploadConflictBehavior.rename)
const conflict_behavior_gps_file: Ref<FileUploadConflictBehavior> = ref(FileUploadConflictBehavior.merge)
const target_rep_names_for_file: Ref<Array<string>> = ref(new Array<string>())
const target_rep_name_for_file: Ref<string> = ref("")
const target_rep_names_for_gps_file: Ref<Array<string>> = ref(new Array<string>())
const target_rep_name_for_gps_file: Ref<string> = ref("")

const gps_log_files: Ref<FileList | null> = ref(null)
const files: Ref<FileList | null> = ref(null)

const uploaded_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())

nextTick(() => load_target_rep_names())

async function load_target_rep_names() {
    target_rep_names_for_file.value.splice(0)
    target_rep_names_for_gps_file.value.splice(0)
    const req = new GetRepositoriesRequest()
    const res = await props.gkill_api.get_repositories(req)
    if (res.errors && res.errors.length != 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length != 0) {
        // emits('received_messages', res.messages)
    }
    res.repositories?.forEach((rep: Repository) => {
        if (rep.type == "directory" && rep.is_enable && rep.use_to_write) {
            if (target_rep_name_for_file.value === "") {
                target_rep_name_for_file.value = rep.rep_name
            }
            target_rep_names_for_file.value.push(rep.rep_name)
        }
        if (rep.type == "gpslog" && rep.is_enable && rep.use_to_write) {
            if (target_rep_name_for_gps_file.value === "") {
                target_rep_name_for_gps_file.value = rep.rep_name
            }
            target_rep_names_for_gps_file.value.push(rep.rep_name)
        }
    })
}

async function upload_files() {
    if (!files.value) {
        return
    }
    progress_upload_file_dialog.value?.show()
    const req = new UploadFilesRequest()
    req.conflict_behavior = conflict_behavior_file.value
    req.target_rep_name = target_rep_name_for_file.value
    req.files = new Array<FileData>()
    for (let i = 0; i < files.value.length; i++) {
        const file = files.value[i]
        const filedata = new FileData()
        filedata.data_base64 = await to_base64(file)
        filedata.file_name = file.name
        req.files.push(filedata)
    }

    const res = await props.gkill_api.upload_files(req)
    if (res.errors && res.errors.length != 0) {
        emits('received_errors', res.errors)
        progress_upload_file_dialog.value?.hide()
        return
    }
    if (res.messages && res.messages.length != 0) {
        emits('received_messages', res.messages)
    }

    uploaded_kyous.value.splice(0)
    for (let i = 0; i < res.uploaded_kyous.length; i++) {
        const uploaded_kyou = res.uploaded_kyous[i]
        await uploaded_kyou.reload()
        uploaded_kyous.value.push(uploaded_kyou)
    }
    files.value = null
    progress_upload_file_dialog.value?.hide()
    decide_related_time_uploaded_file_dialog.value?.show()
}

async function upload_gps_log_files() {
    if (!gps_log_files.value) {
        return
    }
    progress_upload_gps_file_dialog.value?.show()
    const req = new UploadGPSLogFilesRequest()
    req.conflict_behavior = conflict_behavior_gps_file.value
    req.target_rep_name = target_rep_name_for_gps_file.value
    req.gps_log_files = new Array<FileData>()
    for (let i = 0; i < gps_log_files.value.length; i++) {
        const gps_log_file = gps_log_files.value[i]
        const filedata = new FileData()
        filedata.data_base64 = await to_base64(gps_log_file)
        filedata.file_name = gps_log_file.name
        req.gps_log_files.push(filedata)
    }

    const res = await props.gkill_api.upload_gpslog_files(req)
    if (res.errors && res.errors.length != 0) {
        emits('received_errors', res.errors)
        progress_upload_gps_file_dialog.value?.hide()
        return
    }
    if (res.messages && res.messages.length != 0) {
        emits('received_messages', res.messages)
    }
    gps_log_files.value = null
    progress_upload_gps_file_dialog.value?.hide()
}
async function to_base64(file: any): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.readAsDataURL(file);
        reader.onload = () => resolve(reader.result as string);
        reader.onerror = (error) => reject(error);
    })
}

async function reload_kyou(kyou: Kyou) {
    for (let i = 0; i < uploaded_kyous.value.length; i++) {
        const uploaded_kyou = uploaded_kyous.value[i]
        if (kyou.id === uploaded_kyou.id) {
            const updated_kyou = kyou.clone()
            await updated_kyou.reload()
            await updated_kyou.load_all()
            uploaded_kyous.value.splice(i, 1, updated_kyou)
        }
    }
}
</script>