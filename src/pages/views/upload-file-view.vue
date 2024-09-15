<template>
    <DecideRelatedTimeUploadedFileDialog />
    <ProgressUploadFileDialog />
    <ProgressUploadGPSFileDialog />
    <SelectUploadFileTargetRepDialog />
    <SelectUploadGPSFileTargetRepDialog />
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

const props = defineProps<UploadFileViewProps>();
const emits = defineEmits<UploadFileViewEmits>();
const conflict_behavior_file: Ref<UploadFileConflictBehaviorType> = ref(UploadFileConflictBehaviorType.rename);
const conflict_behavior_gps_file: Ref<UploadFileConflictBehaviorType> = ref(UploadFileConflictBehaviorType.overwrite);
const target_rep_name_for_file: Ref<string> = ref("");
const target_rep_name_for_gps_file: Ref<string> = ref("");
</script>
