<template>
    <miSideBarHeader />
    <miKeywordQuery />
    <miBoardQuery />
    <miExtructCheckStateQuery />
    <miSortTypeQuery />
    <miTagQuery />
    <miShareFooter />

</template>
<script setup lang="ts">
import type { FindMiQuery } from '@/classes/api/find_query/find-mi-query';
import miBoardQuery from './mi-board-query.vue';
import miExtructCheckStateQuery from './mi-extruct-check-state-query.vue';
import miKeywordQuery from './mi-keyword-query.vue';
import miShareFooter from './mi-share-footer.vue';
import miSideBarHeader from './mi-side-bar-header.vue';
import miSortTypeQuery from './mi-sort-type-query.vue';
import miTagQuery from './mi-tag-query.vue';
import type { miQueryEditorSidebarEmits } from './mi-query-editor-sidebar-emits';
import type { miQueryEditorSidebarProps } from './mi-query-editor-sidebar-props';

const props = defineProps<miQueryEditorSidebarProps>();
const emits = defineEmits<miQueryEditorSidebarEmits>();
defineExpose(generate_query)

async function generate_query(): Promise<FindMiQuery> {
    throw new Error('Not implemented');
}
</script>