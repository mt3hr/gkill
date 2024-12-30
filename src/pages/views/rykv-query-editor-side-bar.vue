<template>
    <div>
        <v-card class="sidebar_header_wrap background-white pa-0 ma-0" :height="header_height">
            <SidebarHeader class="sidebar_header" :application_config="application_config" :gkill_api="gkill_api"
                :find_kyou_query="query" @requested_search="emits('requested_search', false)"
                :inited="inited_sidebar_header_for_query_sidebar"
                @requested_search_with_update_cache="emits('requested_search', true)"
                @requested_clear_find_query="emits_default_query()" ref="sidebar_header" />
        </v-card>
        <div class="rykv_sidebar">
            <KeywordQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search="emits_current_query()" @request_update_keywords="emits_current_query()"
                @request_update_use_keyword_query="emits_current_query()"
                @request_clear_keyword_query="emits_cleard_keyword_query()"
                :inited="inited_keyword_query_for_query_sidebar" @inited="inited_keyword_query_for_query_sidebar = true"
                ref="keyword_query" />
            <div> <v-divider /> </div>
            <TimeIsQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_timeis_tags="emits_current_query()"
                @request_update_and_search_timeis_word="emits_current_query()"
                @request_update_checked_timeis_tags="emits_current_query()"
                :inited="inited_timeis_query_for_query_sidebar" @inited="inited_timeis_query_for_query_sidebar = true"
                @request_update_timeis_keywords="emits_current_query()"
                @request_update_use_timeis_query="emits_current_query()"
                @request_clear_timeis_query="emits_cleard_timeis_query()" ref="timeis_query" />
            <div> <v-divider /> </div>
            <RepQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_checked_reps="emits_current_query()" @request_clear_rep_query="emits_cleard_rep_query()"
                ref="rep_query" :inited="inited_rep_query_for_query_sidebar"
                @inited="inited_rep_query_for_query_sidebar = true" />
            <div> <v-divider /> </div>
            <TagQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_tags="emits_current_query()"
                @request_update_checked_tags="emits_current_query()" @request_clear_tag_query="emits_cleard_tag_query()"
                ref="tag_query" :inited="inited_tag_query_for_query_sidebar"
                @inited="inited_tag_query_for_query_sidebar = true" />
            <div> <v-divider /> </div>
            <div>
                <CalendarQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                    @request_update_dates="emits_current_query()"
                    @request_update_use_calendar_query="emits_current_query()"
                    @request_clear_calendar_query="emits_cleard_calendar_query()"
                    :inited="inited_calendar_query_for_query_sidebar"
                    @inited="inited_calendar_query_for_query_sidebar = true" ref="calendar_query" />
            </div>
            <div> <v-divider /> </div>
            <MapQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_area="emits_current_query()" @request_update_use_map_query="emits_current_query()"
                @request_clear_map_query="emits_cleard_map_query()" :inited="inited_map_query_for_query_sidebar"
                @inited="inited_map_query_for_query_sidebar = true" ref="map_query" />
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
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import { deepEquals } from '@/classes/deep-equals'

const sidebar_header = ref<InstanceType<typeof SidebarHeader> | null>(null);
const keyword_query = ref<InstanceType<typeof KeywordQuery> | null>(null);
const timeis_query = ref<InstanceType<typeof TimeIsQuery> | null>(null);
const rep_query = ref<InstanceType<typeof RepQuery> | null>(null);
const tag_query = ref<InstanceType<typeof TagQuery> | null>(null);
const calendar_query = ref<InstanceType<typeof CalendarQuery> | null>(null);
const map_query = ref<InstanceType<typeof MapQuery> | null>(null);

const props = defineProps<rykvQueryEditorSidebarProps>()
const emits = defineEmits<rykvQueryEditorSidebarEmits>()
defineExpose({ generate_query, get_default_query })

const header_margin = ref(8)
const header_height: Ref<number> = ref(38 + header_margin.value.valueOf())
const sidebar_height = computed(() => (props.app_content_height.valueOf() - header_height.value).toString().concat("px"))
const header_top_px = computed(() => (props.app_content_height.valueOf() - header_height.value).toString().concat("px"))
const sidebar_top_px = computed(() => (header_height.value * -1).toString().concat("px"))

const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

const is_mounted = ref(false)
nextTick(() => is_mounted.value = true)

const inited = computed(() => {
    if (!is_mounted.value) {
        return false
    }
    return inited_keyword_query_for_query_sidebar.value &&
        inited_timeis_query_for_query_sidebar.value &&
        inited_rep_query_for_query_sidebar.value &&
        inited_tag_query_for_query_sidebar.value &&
        inited_calendar_query_for_query_sidebar.value &&
        inited_map_query_for_query_sidebar.value
})

watch(() => inited.value, (new_value: boolean, old_value: boolean) => {
    if (old_value !== new_value && new_value) {
        default_query.value = generate_query().clone()
        default_query.value.query_id = GkillAPI.get_instance().generate_uuid()
        nextTick(() => { emits('inited') })
    }
})

const inited_sidebar_header_for_query_sidebar = ref(true)
const inited_keyword_query_for_query_sidebar = ref(true)
const inited_timeis_query_for_query_sidebar = ref(false)
const inited_rep_query_for_query_sidebar = ref(false)
const inited_tag_query_for_query_sidebar = ref(false)
const inited_calendar_query_for_query_sidebar = ref(false)
const inited_map_query_for_query_sidebar = ref(false)

watch(() => props.find_kyou_query, (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
    if (deepEquals(new_value, old_value)) {
        return
    }
    query.value = new_value
})

function get_default_query(): FindKyouQuery {
    const query = default_query.value?.clone()
    return query
}

function emits_current_query(): void {
    const currentt_query = generate_query(query.value.query_id)
    emits('updated_query', currentt_query)
}

function generate_query(query_id?: string): FindKyouQuery {
    const find_query = new FindKyouQuery()
    if (query_id) {
        find_query.query_id = query_id
    }

    find_query.is_focus_kyou_in_list_view = props.find_kyou_query ? props.find_kyou_query.is_focus_kyou_in_list_view : false
    find_query.is_image_only_in_sidebar = props.find_kyou_query ? props.find_kyou_query.is_image_only_in_sidebar : false

    if (keyword_query.value) {
        find_query.use_words = keyword_query.value.get_use_words()
        find_query.words_and = keyword_query.value.get_use_word_and_search()
        find_query.keywords = keyword_query.value.get_keywords()
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
        const reps = rep_query.value.get_checked_reps()
        const devices = rep_query.value.get_checked_devices()
        const rep_types = rep_query.value.get_checked_rep_types()
        if (reps) {
            find_query.reps = reps
        }
        if (devices) {
            find_query.devices_in_sidebar = devices
        }
        if (rep_types) {
            find_query.rep_types_in_sidebar = rep_types
        }
    }

    if (tag_query.value) {
        const tags = tag_query.value.get_tags()
        if (tags) {
            find_query.tags = tags
        }
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
        find_query.is_enable_map_circle_in_sidebar = map_query.value.get_is_enable_circle()
    }

    return find_query
}

function emits_cleard_keyword_query(): void {
    const find_query = generate_query()
    find_query.query_id = GkillAPI.get_instance().generate_uuid()
    find_query.use_words = default_query.value.use_words
    find_query.keywords = default_query.value.keywords
    find_query.words_and = default_query.value.words_and
    query.value = find_query
    emits('updated_query_clear', find_query)
}

function emits_cleard_timeis_query(): void {
    const find_query = generate_query()
    find_query.query_id = GkillAPI.get_instance().generate_uuid()
    find_query.use_timeis = default_query.value.use_timeis
    find_query.use_timeis_tags = default_query.value.use_timeis_tags
    find_query.timeis_keywords = default_query.value.timeis_keywords
    find_query.timeis_words_and = default_query.value.timeis_words_and
    find_query.use_timeis_tags = default_query.value.use_timeis_tags
    find_query.timeis_tags = default_query.value.timeis_tags
    find_query.timeis_tags_and = default_query.value.timeis_tags_and
    query.value = find_query
    emits('updated_query_clear', find_query)
}

function emits_cleard_rep_query(): void {
    const find_query = generate_query()
    find_query.query_id = GkillAPI.get_instance().generate_uuid()
    find_query.reps = default_query.value.reps
    find_query.devices_in_sidebar = default_query.value.devices_in_sidebar
    find_query.rep_types_in_sidebar = default_query.value.rep_types_in_sidebar
    query.value = find_query
    emits('updated_query_clear', find_query)
}

function emits_cleard_tag_query(): void {
    const find_query = generate_query()
    find_query.query_id = GkillAPI.get_instance().generate_uuid()
    find_query.tags = default_query.value.tags
    find_query.tags_and = default_query.value.tags_and
    query.value = find_query
    emits('updated_query_clear', find_query)
}

function emits_cleard_map_query(): void {
    const find_query = generate_query()
    find_query.query_id = GkillAPI.get_instance().generate_uuid()
    find_query.use_map = default_query.value.use_map
    find_query.map_latitude = default_query.value.map_latitude
    find_query.map_longitude = default_query.value.map_longitude
    find_query.is_enable_map_circle_in_sidebar = default_query.value.is_enable_map_circle_in_sidebar
    find_query.map_radius = default_query.value.map_radius
    query.value = find_query
    emits('updated_query_clear', find_query)
}

function emits_cleard_calendar_query(): void {
    const find_query = generate_query()
    find_query.query_id = GkillAPI.get_instance().generate_uuid()
    find_query.use_calendar = default_query.value.use_calendar
    find_query.calendar_start_date = default_query.value.calendar_start_date
    find_query.calendar_end_date = default_query.value.calendar_end_date
    query.value = find_query
    emits('updated_query_clear', find_query)
}

function emits_default_query(): void {
    const find_query = default_query.value.clone()
    find_query.query_id = GkillAPI.get_instance().generate_uuid()
    query.value = find_query
    emits('updated_query_clear', find_query)
}
</script>
<style lang="css" scoped>
.sidebar_header_wrap {
    top: v-bind(header_top_px);
    position: sticky;
    border-top: solid 2px #2672ed;
    z-index: 10000;
    border-radius: 0;
}

.sidebar_header {
    position: relative;
    top: calc(v-bind("(header_margin / 2).toString().concat('px')"));
    margin-bottom: calc(v-bind("(header_margin / 2).toString().concat('px')"));
}

.rykv_sidebar {
    min-height: v-bind(sidebar_height);
    top: v-bind(sidebar_top_px);
    position: relative;
}
</style>