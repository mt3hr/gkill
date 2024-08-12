<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { EditTagDialogProps } from './edit-tag-dialog-props';
import { EditTagView } from '../views/edit-tag-view';
import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouView } from '../views/kyou-view.vue';
import { KyouViewEmits } from '../views/kyou-view-emits';

// ˄

export class EditTagDialog {
    // ˅
    
    // ˄

    private props: EditTagDialogProps;

    private kyouView: KyouView;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

    private editTagView: EditTagView;

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