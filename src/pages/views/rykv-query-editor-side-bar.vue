<template>
    <div>
        <SidebarHeader :application_config="application_config" :gkill_api="gkill_api" :query="query"
            @request_search="emits('request_search')" @request_clear_find_query="emits('updated_query', default_query)"
            class="sidebar_header" ref="sidebar_header" />
        <div class="rykv_sidebar">
            <KeywordQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
                @request_update_and_search="emits('updated_query', generate_query())"
                @request_update_keywords="emits('updated_query', generate_query())"
                @request_update_use_keyword_query="emits('updated_query', generate_query())"
                @request_clear_keyword_query="emits('updated_query', generate_cleard_keyword_query())"
                ref="keyword_query" />
            <TimeIsQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
                @request_update_and_search_timeis_tags="emits('updated_query', generate_query())"
                @request_update_and_search_timeis_word="emits('updated_query', generate_query())"
                @request_update_timeis_keywords="emits('updated_query', generate_query())"
                @request_update_use_timeis_query="emits('updated_query', generate_query())"
                @request_clear_timeis_query="emits('updated_query', generate_cleard_timeis_query())"
                ref="timeis_query" />
            <RepQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
                @request_update_checked_reps="emits('updated_query', generate_query())"
                @request_clear_rep_query="emits('updated_query', generate_cleard_rep_query())" ref="rep_query" />
            <TagQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
                @request_update_and_search_tags="emits('updated_query', generate_query())"
                @request_update_checked_tags="emits('updated_query', generate_query())"
                @request_clear_tag_query="emits('updated_query', generate_cleard_tag_query())" ref="tag_query" />
            <CalendarQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
                @request_update_dates="emits('updated_query', generate_query())"
                @request_update_use_calendar_query="emits('updated_query', generate_query())"
                @request_clear_calendar_query="emits('updated_query', generate_cleard_calendar_query())"
                ref="calendar_query" />
            <MapQuery :application_config="application_config" :gkill_api="gkill_api" :query="query"
                @request_update_area="emits('updated_query', generate_query())"
                @request_update_use_map_query="emits('updated_query', generate_query())"
                @request_clear_map_query="emits('updated_query', generate_cleard_map_query())" ref="map_query" />
        </div>
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
import { computed, type Ref, ref, watch } from 'vue'

const sidebar_header = ref<InstanceType<typeof SidebarHeader> | null>(null);
const keyword_query = ref<InstanceType<typeof KeywordQuery> | null>(null);
const timeis_query = ref<InstanceType<typeof TimeIsQuery> | null>(null);
const rep_query = ref<InstanceType<typeof RepQuery> | null>(null);
const tag_query = ref<InstanceType<typeof TagQuery> | null>(null);
const calendar_query = ref<InstanceType<typeof CalendarQuery> | null>(null);
const map_query = ref<InstanceType<typeof MapQuery> | null>(null);

const props = defineProps<rykvQueryEditorSidebarProps>()
const emits = defineEmits<rykvQueryEditorSidebarEmits>()
defineExpose(generate_query)

const header_height: Ref<number> = ref(38)
const sidebar_height = computed(() => (props.app_content_height.valueOf() - header_height.value).toString().concat("px"))
const header_top_px = computed(() => (props.app_content_height.valueOf() - header_height.value).toString().concat("px"))
const sidebar_top_px = computed(() => (header_height.value * -1).toString().concat("px"))

const default_query: Ref<FindKyouQuery> = ref(((): FindKyouQuery => generate_query())())

function generate_query(): FindKyouQuery {
    const find_query = new FindKyouQuery()

    find_query.update_cache = true

    if (keyword_query.value) {
        find_query.use_word = keyword_query.value.get_use_words()
        find_query.keywords = keyword_query.value.get_keywords()
        find_query.words_and = keyword_query.value.get_use_word_and_search()
    }

    if (timeis_query.value) {
        find_query.use_timeis = timeis_query.value.get_use_timeis()
        find_query.timeis_keywords = timeis_query.value.get_timeis_keywords()
        find_query.timeis_words_and = timeis_query.value.get_use_and_search_timeis_words()
        find_query.use_timeis_tags = timeis_query.value.get_use_timeis_tags()
        find_query.timeis_tags = timeis_query.value.get_timeis_tags()
        find_query.timeis_tags_and = timeis_query.value.get_use_and_search_timeis_tags()
    }

    if (rep_query.value) {
        find_query.reps = rep_query.value.get_checked_reps()
    }

    if (tag_query.value) {
        find_query.tags = tag_query.value.get_tags()
        find_query.tags_and = tag_query.value.get_is_and_search()
    }

    if (calendar_query.value) {
        find_query.use_calendar = calendar_query.value.get_use_calendar()
        find_query.calendar_start_date = calendar_query.value.get_start_date()
        find_query.calendar_end_date = calendar_query.value.get_end_date()
    }

    if (map_query.value) {
        find_query.use_map = map_query.value.get_use_map()
        find_query.map_latitude = map_query.value.get_latitude()
        find_query.map_longitude = map_query.value.get_longitude()
        find_query.map_radius = map_query.value.get_radius()
    }

    find_query.parse_words_and_not_words()
    return find_query
}

function generate_cleard_keyword_query(): FindKyouQuery {
    const find_query = generate_query()
    find_query.keywords = default_query.value.keywords
    find_query.words_and = default_query.value.words_and
    find_query.parse_words_and_not_words()
    return find_query
}

function generate_cleard_timeis_query(): FindKyouQuery {
    const find_query = generate_query()
    find_query.timeis_keywords = default_query.value.timeis_keywords
    find_query.timeis_words_and = default_query.value.timeis_words_and
    find_query.use_timeis_tags = default_query.value.use_timeis_tags
    find_query.timeis_tags = default_query.value.timeis_tags
    find_query.timeis_tags_and = default_query.value.timeis_tags_and
    find_query.parse_words_and_not_words()
    return find_query
}

function generate_cleard_rep_query(): FindKyouQuery {
    const find_query = generate_query()
    find_query.reps = default_query.value.reps
    return find_query
}

function generate_cleard_tag_query(): FindKyouQuery {
    const find_query = generate_query()
    find_query.tags = default_query.value.tags
    find_query.tags_and = default_query.value.tags_and
    return find_query
}

function generate_cleard_map_query(): FindKyouQuery {
    const find_query = generate_query()
    find_query.map_latitude = default_query.value.map_latitude
    find_query.map_longitude = default_query.value.map_longitude
    find_query.map_radius = default_query.value.map_radius
    return find_query
}

function generate_cleard_calendar_query(): FindKyouQuery {
    const find_query = generate_query()
    find_query.calendar_start_date = default_query.value.calendar_start_date
    find_query.calendar_end_date = default_query.value.calendar_end_date
    return find_query
}
</script>
<style lang="css">
.sidebar_header {
    top: v-bind(header_top_px);
    position: sticky;
    border-top: solid 2px #2672ed;
}

.rykv_sidebar {
    min-height: v-bind(sidebar_height);
    top: v-bind(sidebar_top_px);
    position: relative;
}
</style>