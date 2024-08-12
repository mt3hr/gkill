<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { AttachedTimeIsPlaingProps } from './attached-time-is-plaing-props';
import { AttachedTimeisPlaingContextMenu } from './attached-timeis-plaing-context-menu';
import { ConfirmDeleteKyouDialog } from '../dialogs/confirm-delete-kyou-dialog.vue';
import { EditTimeIsDialog } from '../dialogs/edit-time-is-dialog.vue';
import { KyouViewEmits } from './kyou-view-emits';

// ˄

export class AttachedTimeIsPlaing {
    // ˅
    
    // ˄

    private cloned_timeis: TimeIs;

    private props: AttachedTimeIsPlaingProps;

    private editTimeIsDialog: EditTimeIsDialog;

    private confirmDeleteKyouDialog: ConfirmDeleteKyouDialog;

    private contextmenu: AttachedTimeisPlaingContextMenu;

    private kyouViewEmits: KyouViewEmits;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>