<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { EditTimeIsDialogProps } from './edit-time-is-dialog-props';
import { EditTimeIsView } from '../views/edit-time-is-view';
import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouView } from '../views/kyou-view.vue';
import { KyouViewEmits } from '../views/kyou-view-emits';

// ˄

export class EditTimeIsDialog {
    // ˅
    
    // ˄

    private props: EditTimeIsDialogProps;

    private kyouView: KyouView;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

    private editTimeIsView: EditTimeIsView;

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