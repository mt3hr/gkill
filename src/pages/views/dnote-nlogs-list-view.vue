<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { DnoteNlogView } from './dnote-nlog-view.vue';
import { DnoteNlogsListViewEmits } from './dnote-nlogs-list-view-emits';
import { DnoteNlogsListViewProps } from './dnote-nlogs-list-view-props';

// ˄

export class DnoteNlogsListView {
    // ˅
    
    // ˄

    private calcrated_total_amount: Ref<Number>;

    private sorted_nlog_kyous: Ref<Array<Kyou>>;

    private props: DnoteNlogsListViewProps;

    private emits: DnoteNlogsListViewEmits;

    private dnoteNlogView: Array<DnoteNlogView>;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>