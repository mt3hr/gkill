<template>
    <div>
        <v-card class="sidebar_header_wrap background-white pa-0 ma-0" :height="header_height">
            <ShareKyouFooter v-if="application_config.is_show_share_footer" class="sidebar_footer"
                :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_open_manage_share_kyou_dialog="show_manage_share_kyou_dialog()"
                @request_open_share_kyou_dialog="show_share_kyou_dialog()"
                @received_messages="(messages: GkillMessage[]) => onReceivedMessages(messages)"
                @received_errors="(errors: GkillError[]) => onReceivedErrors(errors)" />
            <SidebarHeader class="sidebar_header" :application_config="application_config" :gkill_api="gkill_api"
                :find_kyou_query="query" @requested_search="onRequestSearchFalse"
                :inited="inited_sidebar_header_for_query_sidebar"
                @requested_search_with_update_cache="onRequestSearchTrue"
                @requested_clear_find_query="emits_default_query()" ref="sidebar_header" />
        </v-card>
        <div class="mi_sidebar">
            <KeywordQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search="emits_current_query()" @request_update_keywords="emits_current_query()"
                @request_update_use_keyword_query="emits_current_query()"
                @request_clear_keyword_query="emits_cleard_keyword_query()"
                :inited="inited_keyword_query_for_query_sidebar" @inited="onInitedKeyword"
                ref="keyword_query" />
            <div> <v-divider /> </div>
            <miExtructCheckStateQuery :application_config="application_config" :gkill_api="gkill_api"
                :find_kyou_query="query" @request_clear_check_state="emits_cleard_check_state()"
                @request_update_extruct_check_state="emits_current_query()"
                @inited="onInitedCheckState" ref="check_state_query" />
            <div> <v-divider /> </div>
            <miSortTypeQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_sort_type="emits_current_query()"
                @request_clear_sort_type="emits_cleard_sort_type_query()" ref="sort_type_query"
                @inited="onInitedSort" />
            <div> <v-divider /> </div>
            <miBoardQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                :inited="inited_board_query_for_query_sidebar" @inited="onInitedBoard"
                @request_open_focus_board="(board_name: string) => onRequestOpenFocusBoard(board_name)"
                ref="board_query" />
            <div> <v-divider /> </div>
            <TagQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_tags="emits_current_query()"
                @request_update_checked_tags="(_tags, is_by_user) => onTagQueryRequestUpdateCheckedTags(_tags, is_by_user)" @request_clear_tag_query="emits_cleard_tag_query()"
                ref="tag_query" :inited="inited_tag_query_for_query_sidebar"
                @inited="onInitedTag" />
            <div> <v-divider /> </div>
            <TimeIsQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_timeis_tags="emits_current_query()"
                @request_update_and_search_timeis_word="emits_current_query()"
                @request_update_checked_timeis_tags="(tags: string[], is_by_user: boolean) => onTimeisQueryRequestUpdateCheckedTimeisTags(tags, is_by_user)"
                :inited="inited_timeis_query_for_query_sidebar" @inited="onInitedTimeis"
                @request_update_timeis_keywords="emits_current_query()"
                @request_update_use_timeis_query="emits_current_query()"
                @request_clear_timeis_query="emits_cleard_timeis_query()" ref="timeis_query" />
            <div> <v-divider /> </div>
            <PeriodOfTimeQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                :inited="inited_period_of_time_query_for_query_sidebar"
                @request_update_use_period_of_time="emits_current_query()"
                @request_update_period_of_time="emits_current_query()"
                @request_clear_use_period_of_time_query="emits_cleard_period_of_time_query()"
                ref="period_of_time_query" />
            <div> <v-divider /> </div>
            <div>
                <CalendarQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                    @request_update_dates="emits_current_query()"
                    @request_update_use_calendar_query="emits_current_query()"
                    @request_clear_calendar_query="emits_cleard_calendar_query()"
                    :inited="inited_calendar_query_for_query_sidebar"
                    @inited="onInitedCalendar" ref="calendar_query" />
            </div>
            <div> <v-divider /> </div>
            <MapQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_area="emits_current_query()" @request_update_use_map_query="emits_current_query()"
                @request_clear_map_query="emits_cleard_map_query()" :inited="inited_map_query_for_query_sidebar"
                @inited="onInitedMap" ref="map_query" />
        </div>
    </div>
</template>
<script setup lang="ts">
import miBoardQuery from './mi-board-query.vue'
import miExtructCheckStateQuery from './mi-extruct-check-state-query.vue'
import KeywordQuery from './keyword-query.vue'
import CalendarQuery from './calendar-query.vue'
import MapQuery from './map-query.vue'
import SidebarHeader from './sidebar-header.vue'
import TimeIsQuery from './time-is-query.vue'
import TagQuery from './tag-query.vue'
import ShareKyouFooter from './share-kyou-footer.vue'
import miSortTypeQuery from './mi-sort-type-query.vue'
import PeriodOfTimeQuery from './period-of-time-query.vue'
import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { miQueryEditorSidebarEmits } from './mi-query-editor-sidebar-emits'
import type { miQueryEditorSidebarProps } from './mi-query-editor-sidebar-props'
import { useMiQueryEditorSidebar } from '@/classes/use-mi-query-editor-sidebar'

const props = defineProps<miQueryEditorSidebarProps>()
const emits = defineEmits<miQueryEditorSidebarEmits>()

const {
    // Template refs
    sidebar_header,
    keyword_query,
    timeis_query,
    tag_query,
    calendar_query,
    map_query,
    check_state_query,
    sort_type_query,
    board_query,
    period_of_time_query,

    // State
    query,
    inited_sidebar_header_for_query_sidebar,
    inited_keyword_query_for_query_sidebar,
    inited_timeis_query_for_query_sidebar,
    inited_tag_query_for_query_sidebar,
    inited_calendar_query_for_query_sidebar,
    inited_map_query_for_query_sidebar,
    inited_board_query_for_query_sidebar,
    inited_period_of_time_query_for_query_sidebar,

    // Computed
    header_margin,
    header_height,
    sidebar_height,
    header_top_px,
    sidebar_top_px,

    // Exposed methods
    generate_query,
    get_default_query,

    // Template event handlers
    emits_current_query,
    emits_cleard_sort_type_query,
    emits_cleard_check_state,
    emits_cleard_keyword_query,
    emits_cleard_timeis_query,
    emits_cleard_tag_query,
    emits_cleard_map_query,
    emits_cleard_calendar_query,
    emits_cleard_period_of_time_query,
    emits_default_query,
    show_manage_share_kyou_dialog,
    show_share_kyou_dialog,
    onRequestSearchFalse,
    onRequestSearchTrue,
    onRequestOpenFocusBoard,
    onReceivedMessages,
    onReceivedErrors,
    onTagQueryRequestUpdateCheckedTags,
    onTimeisQueryRequestUpdateCheckedTimeisTags,
    onInitedTimeis,
    onInitedTag,
    onInitedCalendar,
    onInitedMap,
    onInitedCheckState,
    onInitedSort,
    onInitedBoard,
    onInitedKeyword,
} = useMiQueryEditorSidebar({ props, emits })

defineExpose({ generate_query, get_default_query })
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

.sidebar_footer {
    position: relative;
    top: calc(v-bind("(header_margin / (2 * 2)).toString().concat('px')"));
}

.mi_sidebar {
    min-height: v-bind(sidebar_height);
    top: v-bind(sidebar_top_px);
    position: relative;
}
</style>
