<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { EditMiDialogProps } from './edit-mi-dialog-props';
import { EditMiView } from '../views/edit-mi-view';
import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouView } from '../views/kyou-view.vue';
import { KyouViewEmits } from '../views/kyou-view-emits';
import { NewBoardNameDialog } from './new-board-name-dialog';

// ˄

export class EditMiDialog {
    // ˅
    
    // ˄

    private new_board_name_dialog: NewBoardNameDialog;

    private props: EditMiDialogProps;

    private kyouView: KyouView;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

    private editMiView: EditMiView;

    async show(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async hide(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>