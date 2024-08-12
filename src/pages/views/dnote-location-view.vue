<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { DnoteLocationViewEmits } from './dnote-location-view-emits';
import { DnoteLocationViewProps } from './dnote-location-view-props';
import { KyouDialog } from '../dialogs/kyou-dialog.vue';

// ˄

export class DnoteLocationView {
    // ˅
    
    // ˄

    private cloned_kyou: Ref<Kyou>;

    private cloned_timeis: Ref<TimeIs>;

    private cloned_kmemo: Ref<Kmemo>;

    private props: DnoteLocationViewProps;

    private emits: DnoteLocationViewEmits;

    private kyou_dialog: KyouDialog;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>