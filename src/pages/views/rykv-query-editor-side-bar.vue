<template>
    <SidebarHeader />
    <KeywordQuery />
    <TimeIsQuery />
    <RepQuery />
    <TagQuery />
    <MapQuery />
    <CalendarQuery />
</template>
<script setup lang="ts">
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import CalendarQuery from './calendar-query.vue';
import KeywordQuery from './keyword-query.vue';
import MapQuery from './map-query.vue';
import RepQuery from './rep-query.vue';
import SidebarHeader from './sidebar-header.vue';
import TagQuery from './tag-query.vue';
import TimeIsQuery from './time-is-query.vue';
import type { rykvQueryEditorSidebarEmits } from './rykv-query-editor-sidebar-emits';
import type { rykvQueryEditorSidebarProps } from './rykv-query-editor-sidebar-props';

const props = defineProps<rykvQueryEditorSidebarProps>();
const emits = defineEmits<rykvQueryEditorSidebarEmits>();
defineExpose(generate_query)

async function generate_query(): Promise<FindKyouQuery> {
    throw new Error('Not implemented');
}
</script>
