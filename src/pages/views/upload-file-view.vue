<template>

</template>
<script setup lang="ts">
// ˅
'use strict';

import { DecideRelatedTimeUploadedFileDialog } from '../dialogs/decide-related-time-uploaded-file-dialog.vue';
import { ProgressUploadFileDialog } from '../dialogs/progress-upload-file-dialog.vue';
import { ProgressUploadGPSFileDialog } from '../dialogs/progress-upload-gps-file-dialog.vue';
import { SelectUploadFileTargetRepDialog } from '../dialogs/select-upload-file-target-rep-dialog.vue';
import { SelectUploadGPSFileTargetRepDialog } from '../dialogs/select-upload-gps-file-target-rep-dialog.vue';
import { SelectUploadGPSFileTargetRepView } from './select-upload-gps-file-target-rep-view.vue';
import { UploadFileViewEmits } from './upload-file-view-emits';
import { UploadFileViewProps } from './upload-file-view-props';

// ˄

export class UploadFileView {
    // ˅
    
    // ˄

    private conflict_behavior_file: Ref<UploadFileConflictBehaviorType>;

    private conflict_behavior_gps_file: Ref<UploadFileConflictBehaviorType>;

    private target_rep_name_for_file: Ref<string>;

    private target_rep_name_for_gps_file: Ref<string>;

    private props: UploadFileViewProps;

    private emits: UploadFileViewEmits;

    private select_upload_file_target_rep_dialog: SelectUploadFileTargetRepDialog;

    private selectUploadGPSFileTargetRepView: SelectUploadGPSFileTargetRepView;

    private select_upload_gpsfile_target_rep_dialog: SelectUploadGPSFileTargetRepDialog;

    private progress_upload_file_dialog: ProgressUploadFileDialog;

    private progress_upload_gps_file_dialog: ProgressUploadGPSFileDialog;

    private decide_related_time_uploaded_file_dialog: DecideRelatedTimeUploadedFileDialog;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>