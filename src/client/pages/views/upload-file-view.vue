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
                        <div
                            class="drop_zone"
                            :class="{ 'drop_zone_active': is_dragging_over_file }"
                            @dragenter.stop="on_dragenter_file"
                            @dragleave.stop="on_dragleave_file"
                            @dragover.prevent.stop="on_dragover_file"
                            @drop.prevent.stop="on_drop_file"
                        >
                            <v-icon size="48">mdi-cloud-upload-outline</v-icon>
                            <div>{{ i18n.global.t('DROP_FILE_ZONE_LABEL') }}</div>
                        </div>
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
                        <div
                            class="drop_zone"
                            :class="{ 'drop_zone_active': is_dragging_over_gps_file }"
                            @dragenter.stop="on_dragenter_gps_file"
                            @dragleave.stop="on_dragleave_gps_file"
                            @dragover.prevent.stop="on_dragover_gps_file"
                            @drop.prevent.stop="on_drop_gps_file"
                        >
                            <v-icon size="48">mdi-cloud-upload-outline</v-icon>
                            <div>{{ i18n.global.t('DROP_GPS_FILE_ZONE_LABEL') }}</div>
                        </div>
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

    // Drag state
    is_dragging_over_file,
    is_dragging_over_gps_file,

    // Drag handlers
    on_dragenter_file,
    on_dragleave_file,
    on_dragover_file,
    on_drop_file,
    on_dragenter_gps_file,
    on_dragleave_gps_file,
    on_dragover_gps_file,
    on_drop_gps_file,

    // Event relay objects
    crudRelayHandlers,
} = useUploadFileView({ props, emits })
</script>
<style scoped>
.drop_zone {
    border: 2px dashed rgba(var(--v-theme-primary), 0.4);
    border-radius: 8px;
    padding: 32px 16px;
    text-align: center;
    margin: 8px;
    transition: background-color 0.2s ease, border-color 0.2s ease;
    user-select: none;
}

.drop_zone * {
    pointer-events: none;
}

.drop_zone_active {
    background-color: rgba(var(--v-theme-primary), 0.1);
    border-color: rgb(var(--v-theme-primary));
    border-style: solid;
}
</style>
