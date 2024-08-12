<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { AttachedTextContextMenu } from './attached-text-context-menu.vue';
import { AttachedTextProps } from './attached-text-props';
import { ConfirmDeleteTextDialog } from '../dialogs/confirm-delete-text-dialog.vue';
import { EditTextDialog } from '../dialogs/edit-text-dialog.vue';
import { KyouViewEmits } from './kyou-view-emits';

// ˄

export class AttachedText {
    // ˅
    
    // ˄

    private cloned_text: Text;

    private props: AttachedTextProps;

    private editTextDialog: EditTextDialog;

    private confirmDeleteTextDialog: ConfirmDeleteTextDialog;

    private contextmenu: AttachedTextContextMenu;

    private emits: KyouViewEmits;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>