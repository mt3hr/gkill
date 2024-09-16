<template>
    <DecideRelatedTimeUploadedFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api" :uploaded_kyous="uploaded_kyous"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <ProgressUploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    />
    <ProgressUploadGPSFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    />
    <SelectUploadFileTargetRepDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    />
    <SelectUploadGPSFileTargetRepDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import { UploadFileConflictBehaviorType } from '@/classes/api/upload-file-conflict-behavior-type';
import DecideRelatedTimeUploadedFileDialog from '../dialogs/decide-related-time-uploaded-file-dialog.vue';
import ProgressUploadFileDialog from '../dialogs/progress-upload-file-dialog.vue';
import ProgressUploadGPSFileDialog from '../dialogs/progress-upload-gps-file-dialog.vue';
import SelectUploadFileTargetRepDialog from '../dialogs/select-upload-file-target-rep-dialog.vue';
import SelectUploadGPSFileTargetRepDialog from '../dialogs/select-upload-gps-file-target-rep-dialog.vue';

import type { UploadFileViewEmits } from './upload-file-view-emits';
import type { UploadFileViewProps } from './upload-file-view-props';
import type { Kyou } from '@/classes/datas/kyou';

const props = defineProps<UploadFileViewProps>();
const emits = defineEmits<UploadFileViewEmits>();
const conflict_behavior_file: Ref<UploadFileConflictBehaviorType> = ref(UploadFileConflictBehaviorType.rename);
const conflict_behavior_gps_file: Ref<UploadFileConflictBehaviorType> = ref(UploadFileConflictBehaviorType.overwrite);
const target_rep_name_for_file: Ref<string> = ref("");
const target_rep_name_for_gps_file: Ref<string> = ref("");

const uploaded_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
</script>
