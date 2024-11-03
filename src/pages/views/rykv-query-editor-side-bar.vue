<template>
    <div class="rykv_sidebar">
        <SidebarHeader :application_config="application_config" :gkill_api="gkill_api" :query="query"
            @request_search="emits('request_search')"
            @request_clear_find_query="emits('updated_query', default_query)" />
        <KeywordQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
            @request_update_and_search="emits('updated_query', generate_query())"
            @request_update_keywords="emits('updated_query', generate_query())"
            @request_update_use_keyword_query="emits('updated_query', generate_query())"
            @request_clear_keyword_query="emits('updated_query', generate_cleard_keyword_query())" />
        <TimeIsQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
            @request_update_and_search_timeis_tags="emits('updated_query', generate_query())"
            @request_update_and_search_timeis_word="emits('updated_query', generate_query())"
            @request_update_timeis_keywords="emits('updated_query', generate_query())"
            @request_update_use_timeis_query="emits('updated_query', generate_query())"
            @request_clear_timeis_query="emits('updated_query', generate_cleard_timeis_query())" />
        <RepQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
            @request_update_checked_reps="emits('updated_query', generate_query())"
            @request_clear_rep_query="emits('updated_query', generate_cleard_rep_query())" />
        <TagQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
            @request_update_and_search_tags="emits('updated_query', generate_query())"
            @request_update_checked_tags="emits('updated_query', generate_query())"
            @request_clear_tag_query="emits('updated_query', generate_cleard_tag_query())" />
        <MapQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
            @request_update_area="emits('updated_query', generate_query())"
            @request_update_use_map_query="emits('updated_query', generate_query())"
            @request_clear_map_query="emits('updated_query', generate_cleard_map_query())" />
        <CalendarQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
            @request_update_dates="emits('updated_query', generate_query())"
            @request_update_use_calendar_query="emits('updated_query', generate_query())"
            @request_clear_calendar_query="emits('updated_query', generate_cleard_calendar_query())" />
    </div>
</template>
<script setup lang="ts">
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import CalendarQuery from './calendar-query.vue'
import KeywordQuery from './keyword-query.vue'
import MapQuery from './map-query.vue'
import RepQuery from './rep-query.vue'
import SidebarHeader from './sidebar-header.vue'
import TagQuery from './tag-query.vue'
import TimeIsQuery from './time-is-query.vue'
import type { rykvQueryEditorSidebarEmits } from './rykv-query-editor-sidebar-emits'
import type { rykvQueryEditorSidebarProps } from './rykv-query-editor-sidebar-props'
import { type Ref, ref } from 'vue'

const props = defineProps<rykvQueryEditorSidebarProps>()
const emits = defineEmits<rykvQueryEditorSidebarEmits>()
defineExpose(generate_query)

const default_query: Ref<FindKyouQuery> = ref(function (): FindKyouQuery {
    //TODO
    return new FindKyouQuery()
}())

function generate_query(): FindKyouQuery {
    throw new Error('Not implemented')
}

function generate_cleard_keyword_query(): FindKyouQuery {
    throw new Error('Not implemented')
}

function generate_cleard_timeis_query(): FindKyouQuery {
    throw new Error('Not implemented')
}

function generate_cleard_rep_query(): FindKyouQuery {
    throw new Error('Not implemented')
}

function generate_cleard_tag_query(): FindKyouQuery {
    throw new Error('Not implemented')
}

function generate_cleard_map_query(): FindKyouQuery {
    throw new Error('Not implemented')
}

function generate_cleard_calendar_query(): FindKyouQuery {
    throw new Error('Not implemented')
}
</script>