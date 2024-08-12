<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { miBoardQuery } from './mi-board-query.vue';
import { miExtructCheckStateQuery } from './mi-extruct-check-state-query.vue';
import { miKeywordQuery } from './mi-keyword-query.vue';
import { miQueryEditorSidebarEmits } from './mi-query-editor-sidebar-emits';
import { miQueryEditorSidebarProps } from './mi-query-editor-sidebar-props';
import { miShareFooter } from './mi-share-footer';
import { miSideBarHeader } from './mi-side-bar-header';
import { miSortTypeQuery } from './mi-sort-type-query';
import { miTagQuery } from './mi-tag-query';

// ˄

export class miQueryEditorSidebar {
    // ˅
    
    // ˄

    private props: miQueryEditorSidebarProps;

    private emits: miQueryEditorSidebarEmits;

    private header: miSideBarHeader;

    private keyword_query: miKeywordQuery;

    private board_query: miBoardQuery;

    private extruct_check_state_query: miExtructCheckStateQuery;

    private sort_type_query: miSortTypeQuery;

    private tag_query: miTagQuery;

    private footer: miShareFooter;

    async generate_query(): Promise<FindMiQuery> {
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