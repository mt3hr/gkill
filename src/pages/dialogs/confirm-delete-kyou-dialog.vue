<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { ConfirmDeleteKyouDialogProps } from './confirm-delete-kyou-dialog-props';
import { ConfirmDeleteKyouView } from '../views/confirm-delete-kyou-view';
import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouView } from '../views/kyou-view.vue';
import { KyouViewEmits } from '../views/kyou-view-emits';

// ˄

export class ConfirmDeleteKyouDialog {
    // ˅
    
    // ˄

    private props: ConfirmDeleteKyouDialogProps;

    private kyouView: KyouView;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

    private confirmDeleteKyouView: ConfirmDeleteKyouView;

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