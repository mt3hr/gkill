<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { EditIDFKyouDialogProps } from './edit-idf-kyou-dialog-props';
import { EditIDFKyouView } from '../views/edit-idf-kyou-view';
import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouView } from '../views/kyou-view.vue';
import { KyouViewEmits } from '../views/kyou-view-emits';

// ˄

export class EditIDFKyouDialog {
    // ˅
    
    // ˄

    private props: EditIDFKyouDialogProps;

    private kyouView: KyouView;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

    private editIDFKyouView: EditIDFKyouView;

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