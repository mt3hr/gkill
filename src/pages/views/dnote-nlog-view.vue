<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { DnoteNlogViewEmits } from './dnote-nlog-view-emits';
import { DnoteNlogViewProps } from './dnote-nlog-view-props';
import { KyouDialog } from '../dialogs/kyou-dialog.vue';

// ˄

export class DnoteNlogView {
    // ˅
    
    // ˄

    private cloned_kyou: Ref<Kyou>;

    private cloned_nlog: Ref<Nlog>;

    private props: DnoteNlogViewProps;

    private emits: DnoteNlogViewEmits;

    private kyou_dialog: KyouDialog;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>