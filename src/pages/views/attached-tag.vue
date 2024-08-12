<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { AttachedTagContextMenu } from './attached-tag-context-menu.vue';
import { AttachedTagProps } from './attached-tag-props';
import { ConfirmDeleteTagDialog } from '../dialogs/confirm-delete-tag-dialog.vue';
import { EditTagDialog } from '../dialogs/edit-tag-dialog.vue';
import { KyouViewEmits } from './kyou-view-emits';

// ˄

export class AttachedTag {
    // ˅
    
    // ˄

    private cloned_tag: Tag;

    private props: AttachedTagProps;

    private editTagDialog: EditTagDialog;

    private confirmDeleteTagDialog: ConfirmDeleteTagDialog;

    private contextmenu: AttachedTagContextMenu;

    private emits: KyouViewEmits;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>