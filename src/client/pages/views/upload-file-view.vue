<template>
    <div v-show="target_rep_name_for_file !== '' && target_rep_name_for_gps_file !== ''">
        <v-card>
            <v-card-title>
                {{ i18n.global.t("UPLOAD_FILE_TITLE") }}
            </v-card-title>
            <v-tabs v-model="tab">
                <v-tab key="file">{{ i18n.global.t("FILE_TITLE") }}</v-tab>
                <v-tab key="gps_log_file">{{ i18n.global.t("GPS_LOG_FILE_TITLE") }}</v-tab>
            </v-tabs>
            <v-window v-model="tab">
                <v-window-item key="file" :eager="true">
                    <v-card>
                        <v-card-title>
                            {{ i18n.global.t("FILE_NAME_COLLISION_TITLE") }}
                        </v-card-title>
                        <v-radio-group v-model="conflict_behavior_file">
                            <v-radio :label="i18n.global.t('RENAME_TITLE')" :value="'rename'" />
                            <v-radio :label="i18n.global.t('OVERRIDE_TITLE')" :value="'override'" />
                        </v-radio-group>
                        <v-select class="select" v-model="target_rep_name_for_file" :items="target_rep_names_for_file"
                            :label="i18n.global.t('UPLOAD_DESTINATION_TITLE')" />
                        <v-file-input :label="i18n.global.t('FILE_TITLE')" multiple @change="upload_files"
                            v-model="files" />
                    </v-card>
                </v-window-item>
                <v-window-item key="gps_log_file" :eager="true">
                    <v-card>
                        <v-card-title>
                            {{ i18n.global.t("FILE_NAME_COLLISION_TITLE") }}
                        </v-card-title>
                        <v-radio-group v-model="conflict_behavior_gps_file">
                            <v-radio :label="i18n.global.t('MERGE_TITLE')" :value="'merge'" />
                            <v-radio :label="i18n.global.t('OVERRIDE_TITLE')" :value="'override'" />
                        </v-radio-group>
                        <v-select class="select" v-model="target_rep_name_for_gps_file"
                            :items="target_rep_names_for_gps_file" :label="i18n.global.t('UPLOAD_DESTINATION_TITLE')" />
                        <v-file-input :label="i18n.global.t('GPS_LOG_FILE_TITLE')" multiple
                            @change="upload_gps_log_files" v-model="gps_log_files" accept=".gpx" />
                    </v-card>
                </v-window-item>
            </v-window>
        </v-card>
        <DecideRelatedTimeUploadedFileDialog :app_content_height="app_content_height"
            :app_content_width="app_content_width" :application_config="application_config" :gkill_api="gkill_api"
            :uploaded_kyous="uploaded_kyous"
            v-on="crudRelayHandlers"
            ref="decide_related_time_uploaded_file_dialog" />
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import DecideRelatedTimeUploadedFileDialog from '../dialogs/decide-related-time-uploaded-file-dialog.vue'
import type { UploadFileViewProps } from './upload-file-view-props'
import type { UploadFileViewEmits } from './upload-file-view-emits'
import { useUploadFileView } from '@/classes/use-upload-file-view'

const props = defineProps<UploadFileViewProps>()
const emits = defineEmits<UploadFileViewEmits>()

const {
    // Template refs
    decide_related_time_uploaded_file_dialog,

    // State
    tab,
    conflict_behavior_file,
    conflict_behavior_gps_file,
    target_rep_names_for_file,
    target_rep_name_for_file,
    target_rep_names_for_gps_file,
    target_rep_name_for_gps_file,
    gps_log_files,
    files,
    uploaded_kyous,

    // Business logic / template handlers
    upload_files,
    upload_gps_log_files,

    // Event relay objects
    crudRelayHandlers,
} = useUploadFileView({ props, emits })
</script>
