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
            <v-avatar v-if="saved_find_querys.length > 0" color="primary" class="saved_find_query_fab">
                <v-menu transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" icon="mdi-bookmark-multiple" variant="text" v-bind="props"
                            :title="i18n.global.t('OPEN_SAVED_FIND_QUERY_LIST_TITLE')" />
                    </template>
                    <v-list>
                        <v-list-item v-for="item in saved_find_querys" :key="item.id" @click="apply_saved_query(item)">
                            <v-list-item-title>{{ item.title }}</v-list-item-title>
                        </v-list-item>
                    </v-list>
                </v-menu>
            </v-avatar>
        </v-card>
        <div class="mi_sidebar">
            <KeywordQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search="emits_current_query()" @request_update_keywords="emits_current_query()"
                @request_update_use_keyword_query="emits_current_query()"
                @request_clear_keyword_query="emits_cleard_keyword_query()"
                :inited="inited_keyword_query_for_query_sidebar" @inited="onInitedKeyword"
                ref="keyword_query" />
            <div> <v-divider /> </div>
            <miExtractCheckStateQuery :application_config="application_config" :gkill_api="gkill_api"
                :find_kyou_query="query" @request_clear_check_state="emits_cleard_check_state()"
                @request_update_extract_check_state="emits_current_query()"
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
            <div>
                <CalendarQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                    @request_update_dates="emits_current_query()"
                    @request_update_use_calendar_query="emits_current_query()"
                    @request_clear_calendar_query="emits_cleard_calendar_query()"
                    :inited="inited_calendar_query_for_query_sidebar"
                    @inited="onInitedCalendar" ref="calendar_query" />
            </div>
        </div>
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import miBoardQuery from './mi-board-query.vue'
import miExtractCheckStateQuery from './mi-extract-check-state-query.vue'
import KeywordQuery from './keyword-query.vue'
import CalendarQuery from './calendar-query.vue'
import SidebarHeader from './sidebar-header.vue'
import TagQuery from './tag-query.vue'
import ShareKyouFooter from './share-kyou-footer.vue'
import miSortTypeQuery from './mi-sort-type-query.vue'
import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { MiQueryEditorSidebarEmits } from './mi-query-editor-sidebar-emits'
import type { MiQueryEditorSidebarProps } from './mi-query-editor-sidebar-props'
import { useMiQueryEditorSidebar } from '@/classes/use-mi-query-editor-sidebar'

const props = defineProps<MiQueryEditorSidebarProps>()
const emits = defineEmits<MiQueryEditorSidebarEmits>()

const {
    // Template refs
    sidebar_header,
    keyword_query,
    tag_query,
    calendar_query,
    check_state_query,
    sort_type_query,
    board_query,

    // State
    query,
    inited_sidebar_header_for_query_sidebar,
    inited_keyword_query_for_query_sidebar,
    inited_tag_query_for_query_sidebar,
    inited_calendar_query_for_query_sidebar,
    inited_board_query_for_query_sidebar,

    // Computed
    header_margin,
    header_height,
    sidebar_height,
    header_top_px,
    sidebar_top_px,
    saved_find_querys,

    // Exposed methods
    generate_query,
    get_default_query,
    apply_saved_query,

    // Template event handlers
    emits_current_query,
    emits_cleard_sort_type_query,
    emits_cleard_check_state,
    emits_cleard_keyword_query,
    emits_cleard_tag_query,
    emits_cleard_calendar_query,
    emits_default_query,
    show_manage_share_kyou_dialog,
    show_share_kyou_dialog,
    onRequestSearchFalse,
    onRequestSearchTrue,
    onRequestOpenFocusBoard,
    onReceivedMessages,
    onReceivedErrors,
    onTagQueryRequestUpdateCheckedTags,
    onInitedTag,
    onInitedCalendar,
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
    /* 保存済み検索条件FABをバーの上へはみ出させるため(v-cardの既定はhidden) */
    overflow: visible;
}

.saved_find_query_fab {
    position: absolute;
    top: -60px;
    right: 10px;
    height: 50px;
    width: 50px;
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
