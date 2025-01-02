<template>
    <div>
        <v-card class="sidebar_header_wrap background-white pa-0 ma-0" :height="header_height">
            <miShareFooter class="sidebar_footer" :application_config="application_config" :gkill_api="gkill_api"
                :find_kyou_query="query" @request_open_manage_share_mi_dialog="show_manage_share_mi_dialog()"
                @request_open_share_mi_dialog="show_share_mi_dialog()"
                @received_messages="(messages) => emits('received_messages', messages)"
                @received_errors="(errors) => emits('received_errors', errors)" />
            <SidebarHeader class="sidebar_header" :application_config="application_config" :gkill_api="gkill_api"
                :find_kyou_query="query" @requested_search="emits('requested_search', false)"
                :inited="inited_sidebar_header_for_query_sidebar"
                @requested_search_with_update_cache="emits('requested_search', true)"
                @requested_clear_find_query="emits_default_query()" ref="sidebar_header" />
        </v-card>
        <div class="mi_sidebar">
            <KeywordQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search="emits_current_query()" @request_update_keywords="emits_current_query()"
                @request_update_use_keyword_query="emits_current_query()"
                @request_clear_keyword_query="emits_cleard_keyword_query()"
                :inited="inited_keyword_query_for_query_sidebar" @inited="inited_keyword_query_for_query_sidebar = true"
                ref="keyword_query" />
            <div> <v-divider /> </div>
            <miExtructCheckStateQuery :application_config="application_config" :gkill_api="gkill_api"
                :find_kyou_query="query" @request_clear_check_state="emits_cleard_check_state()"
                @request_update_extruct_check_state="emits_current_query()"
                @inited="inited_check_state_query_for_query_sidebar = true" ref="check_state_query" />
            <div> <v-divider /> </div>
            <miSortTypeQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_sort_type="emits_current_query()"
                @request_clear_sort_type="emits_cleard_sort_type_query()" ref="sort_type_query"
                @inited="inited_sort_query_for_query_sidebar = true" />
            <div> <v-divider /> </div>
            <miBoardQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                :inited="inited_board_query_for_query_sidebar" @inited="inited_board_query_for_query_sidebar = true"
                @request_open_focus_board="(board_name) => emits('request_open_focus_board', board_name)"
                ref="board_query" />
            <div> <v-divider /> </div>
            <TagQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_tags="emits_current_query()"
                @request_update_checked_tags="emits_current_query()" @request_clear_tag_query="emits_cleard_tag_query()"
                ref="tag_query" :inited="inited_tag_query_for_query_sidebar"
                @inited="inited_tag_query_for_query_sidebar = true" />
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
import miBoardQuery from './mi-board-query.vue'
import miExtructCheckStateQuery from './mi-extruct-check-state-query.vue'
import KeywordQuery from './keyword-query.vue'
import CalendarQuery from './calendar-query.vue'
import MapQuery from './map-query.vue'
import RepQuery from './rep-query.vue'
import SidebarHeader from './sidebar-header.vue'
import TimeIsQuery from './time-is-query.vue'
import TagQuery from './tag-query.vue'
import miShareFooter from './mi-share-footer.vue'
import miSortTypeQuery from './mi-sort-type-query.vue'
import type { miQueryEditorSidebarEmits } from './mi-query-editor-sidebar-emits'
import type { miQueryEditorSidebarProps } from './mi-query-editor-sidebar-props'
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { GkillAPI } from '@/classes/api/gkill-api'
import { deepEquals } from '@/classes/deep-equals'
import type { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type MiBoardQuery from './mi-board-query.vue'

const sidebar_header = ref<InstanceType<typeof SidebarHeader> | null>(null);
const keyword_query = ref<InstanceType<typeof KeywordQuery> | null>(null);
const timeis_query = ref<InstanceType<typeof TimeIsQuery> | null>(null);
const tag_query = ref<InstanceType<typeof TagQuery> | null>(null);
const calendar_query = ref<InstanceType<typeof CalendarQuery> | null>(null);
const map_query = ref<InstanceType<typeof MapQuery> | null>(null);
const check_state_query = ref<InstanceType<typeof miExtructCheckStateQuery> | null>(null);
const sort_type_query = ref<InstanceType<typeof miSortTypeQuery> | null>(null);
const board_query = ref<InstanceType<typeof MiBoardQuery> | null>(null);

const props = defineProps<miQueryEditorSidebarProps>()
const emits = defineEmits<miQueryEditorSidebarEmits>()
defineExpose({ generate_query, get_default_query })

const header_margin = ref(12)
const header_height: Ref<number> = ref(36 * 2 + header_margin.value.valueOf())
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
        inited_tag_query_for_query_sidebar.value &&
        inited_calendar_query_for_query_sidebar.value &&
        inited_map_query_for_query_sidebar.value &&
        inited_check_state_query_for_query_sidebar.value &&
        inited_sort_query_for_query_sidebar.value &&
        inited_board_query_for_query_sidebar.value
})

watch(() => inited.value, async (new_value: boolean, old_value: boolean) => {
    if (old_value !== new_value && new_value) {
        default_query.value = generate_query().clone()
        default_query.value.query_id = props.gkill_api.generate_uuid()
        default_query.value.use_mi_board_name = false
        default_query.value.mi_board_name = "すべて"
        default_query.value.mi_board_name = ""
        nextTick(() => { emits('inited') })
    }
})

const inited_sidebar_header_for_query_sidebar = ref(true)
const inited_keyword_query_for_query_sidebar = ref(true)
const inited_timeis_query_for_query_sidebar = ref(false)
const inited_tag_query_for_query_sidebar = ref(false)
const inited_calendar_query_for_query_sidebar = ref(false)
const inited_map_query_for_query_sidebar = ref(false)
const inited_check_state_query_for_query_sidebar = ref(false)
const inited_sort_query_for_query_sidebar = ref(false)
const inited_board_query_for_query_sidebar = ref(false)

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
    find_query.for_mi = true

    find_query.is_focus_kyou_in_list_view = props.find_kyou_query ? props.find_kyou_query.is_focus_kyou_in_list_view : false
    find_query.is_image_only = props.find_kyou_query ? props.find_kyou_query.is_image_only : false

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

    if (board_query.value) {
        find_query.mi_board_name = board_query.value.get_board_name()
        if (find_query.mi_board_name !== "すべて") {
            find_query.use_mi_board_name = true
        } else {
            find_query.use_mi_board_name = false
        }
    }

    if (true/*rep_query.value*/) {
        const reps = new Array<string>()
        for (let i = 0; i < props.application_config.rep_struct.length; i++) {
            reps.push(props.application_config.rep_struct[i].rep_name)
        }
        find_query.reps = reps
        find_query.use_rep_types = true
        find_query.rep_types = ["mi"]
    }

    if (check_state_query.value) {
        find_query.mi_check_state = check_state_query.value.get_update_extruct_check_state()
    }

    if (sort_type_query.value) {
        find_query.mi_sort_type = sort_type_query.value.get_sort_type()
        find_query.include_create_mi = true
        switch (find_query.mi_sort_type) {
            case MiSortType.create_time:
                break
            case MiSortType.estimate_end_time:
                find_query.include_end_mi = true
                break
            case MiSortType.estimate_start_time:
                find_query.include_start_mi = true
                break
            case MiSortType.limit_time:
                find_query.include_limit_mi = true
                break
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

function emits_cleard_sort_type_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.mi_sort_type = default_query.value.mi_sort_type
    emits('updated_query_clear', find_query)
}

function emits_cleard_check_state(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.mi_check_state = default_query.value.mi_check_state
    emits('updated_query_clear', find_query)
}

function emits_cleard_keyword_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.use_words = default_query.value.use_words
    find_query.keywords = default_query.value.keywords
    find_query.words_and = default_query.value.words_and
    query.value = find_query
    emits('updated_query_clear', find_query)
}

function emits_cleard_timeis_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
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

function emits_cleard_tag_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.tags = default_query.value.tags
    find_query.tags_and = default_query.value.tags_and
    query.value = find_query
    emits('updated_query_clear', find_query)
}

function emits_cleard_map_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
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
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.use_calendar = default_query.value.use_calendar
    find_query.calendar_start_date = default_query.value.calendar_start_date
    find_query.calendar_end_date = default_query.value.calendar_end_date
    query.value = find_query
    emits('updated_query_clear', find_query)
}

function emits_default_query(): void {
    const find_query = default_query.value.clone()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.use_mi_board_name = false
    query.value = find_query
    emits('updated_query_clear', find_query)
}

async function show_manage_share_mi_dialog(): Promise<void> {
    throw new Error('Not implemented')
}

async function show_share_mi_dialog(): Promise<void> {
    throw new Error('Not implemented')
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
    margin-top: calc(v-bind("(header_margin * 1 / 3).toString().concat('px')")) !important;
    margin-bottom: calc(v-bind("(header_margin * 1 / 3).toString().concat('px')")) !important;
}

.sidebar_footer {
    position: relative;
    margin-top: calc(v-bind("(header_margin * 1 / 3).toString().concat('px')")) !important;
    margin-bottom: calc(v-bind("(header_margin * 1 / 3).toString().concat('px')")) !important;
}

.mi_sidebar {
    min-height: v-bind(sidebar_height);
    top: v-bind(sidebar_top_px);
    position: relative;
}
</style>