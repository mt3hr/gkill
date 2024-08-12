<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { AddTextDialogProps } from './add-text-dialog-props';
import { AddTextView } from '../views/add-text-view';
import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouView } from '../views/kyou-view.vue';
import { KyouViewEmits } from '../views/kyou-view-emits';

// ˄

export class AddTextDialog {
    // ˅
    
    // ˄

    private props: AddTextDialogProps;

    private kyouView: KyouView;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

    private addTextView: AddTextView;

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