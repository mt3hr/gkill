<template>
    <miSideBarHeader :application_config="application_config" :board_names="board_names" :gkill_api="gkill_api"
        :query="query" :sort_type="sort_type" @request_clear_find_query="emits('updated_query', default_query)"
        @request_search="emits('request_search')" />
    <miKeywordQuery :application_config="application_config" :board_names="board_names" :gkill_api="gkill_api"
        :query="query" :sort_type="sort_type"
        @request_clear_keyword_query="emits('updated_query', generate_cleard_keyword_query())"
        @request_update_and_search="emits('updated_query', generate_query())"
        @request_update_keywords="emits('updated_query', generate_query())"
        @request_update_use_keyword_query="emits('updated_query', generate_query())" />
    <miBoardQuery :application_config="application_config" :board_names="board_names" :gkill_api="gkill_api"
        :query="query" :sort_type="sort_type"
        @request_open_focus_board="(board_name) => emits('request_open_focus_board', board_name)" />
    <miExtructCheckStateQuery :application_config="application_config" :board_names="board_names" :gkill_api="gkill_api"
        :query="query" :sort_type="sort_type"
        @request_update_extruct_check_state="emits('updated_query', generate_query())" />
    <miSortTypeQuery :application_config="application_config" :board_names="board_names" :gkill_api="gkill_api"
        :query="query" :sort_type="sort_type" @request_update_sort_type="emits('updated_query', generate_query())" />
    <miTagQuery :application_config="application_config" :board_names="board_names" :gkill_api="gkill_api"
        :query="query" :sort_type="sort_type"
        @request_clear_tag_query="() => emits('updated_query', generate_cleard_tag_query())"
        @request_update_and_search_tags="emits('updated_query', generate_query())"
        @request_update_checked_tags="emits('updated_query', generate_query())" />
    <miShareFooter :application_config="application_config" :board_names="board_names" :gkill_api="gkill_api"
        :query="query" :sort_type="sort_type" @request_open_manage_share_mi_dialog="show_manage_share_mi_dialog()"
        @request_open_share_mi_dialog="show_share_mi_dialog()" />
</template>
<script setup lang="ts">
import { FindMiQuery } from '@/classes/api/find_query/find-mi-query'
import miBoardQuery from './mi-board-query.vue'
import miExtructCheckStateQuery from './mi-extruct-check-state-query.vue'
import miKeywordQuery from './mi-keyword-query.vue'
import miShareFooter from './mi-share-footer.vue'
import miSideBarHeader from './mi-side-bar-header.vue'
import miSortTypeQuery from './mi-sort-type-query.vue'
import miTagQuery from './mi-tag-query.vue'
import type { miQueryEditorSidebarEmits } from './mi-query-editor-sidebar-emits'
import type { miQueryEditorSidebarProps } from './mi-query-editor-sidebar-props'
import { type Ref, ref } from 'vue'

const props = defineProps<miQueryEditorSidebarProps>()
const emits = defineEmits<miQueryEditorSidebarEmits>()
defineExpose(generate_query)

const default_query: Ref<FindMiQuery> = ref(function (): FindMiQuery {
    //TODO
    return new FindMiQuery()
}())


function generate_query(): FindMiQuery {
    throw new Error('Not implemented')
}

function generate_cleard_keyword_query(): FindMiQuery {
    throw new Error('Not implemented')
}

function generate_cleard_tag_query(): FindMiQuery {
    throw new Error('Not implemented')
}

async function show_manage_share_mi_dialog(): Promise<void> {
    throw new Error('Not implemented')
}

async function show_share_mi_dialog(): Promise<void> {
    throw new Error('Not implemented')
}
</script>
