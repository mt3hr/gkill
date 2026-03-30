<template>
    <v-card class="find_query_editor_view dummy" :class="loading_class">
        <div class="overlay_target">
            <v-overlay v-model="loading" class="align-center justify-center" persistent contained>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
        </div>
        <div v-show="inited" class="find_query_editor_view">
            <KeywordQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search="emits_current_query()" @request_update_keywords="emits_current_query()"
                @request_update_use_keyword_query="emits_current_query()"
                @request_clear_keyword_query="emits_cleard_keyword_query()"
                :inited="inited_keyword_query_for_query_sidebar" @inited="onInitedKeyword"
                ref="keyword_query" />
            <div> <v-divider /> </div>
            <RepQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_checked_reps="(_reps, is_by_user) => onRepQueryRequestUpdateCheckedReps(_reps, is_by_user)"
                @request_clear_rep_query="emits_cleard_rep_query()" ref="rep_query"
                :inited="inited_rep_query_for_query_sidebar" @inited="onInitedRep" />
            <div> <v-divider /> </div>
            <TagQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_tags="emits_current_query()"
                @request_update_checked_tags="(_tags, is_by_user) => onTagQueryRequestUpdateCheckedTags(_tags, is_by_user)"
                @request_clear_tag_query="emits_cleard_tag_query()" ref="tag_query"
                :inited="inited_tag_query_for_query_sidebar" @inited="onInitedTag" />
            <div> <v-divider /> </div>
            <TimeIsQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_timeis_tags="emits_current_query()"
                @request_update_and_search_timeis_word="emits_current_query()"
                @request_update_checked_timeis_tags="(tags: string[]) => onTimeisQueryRequestUpdateCheckedTimeisTags(tags, true)"
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
            <MapQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_area="emits_current_query()" @request_update_use_map_query="emits_current_query()"
                @request_clear_map_query="emits_cleard_map_query()" :inited="inited_map_query_for_query_sidebar"
                @inited="onInitedMap" ref="map_query" />
        </div>
        <v-card-action v-if="!loading">
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary"
                        @click="onSaveClicked">{{
                            i18n.global.t("SAVE_TITLE") }}</v-btn>
                </v-col>
                <v-spacer class="pa-0 ma-0" />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits_default_query()">{{
                        i18n.global.t("CLEAR_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import KeywordQuery from './keyword-query.vue'
import MapQuery from './map-query.vue'
import RepQuery from './rep-query.vue'
import TagQuery from './tag-query.vue'
import TimeIsQuery from './time-is-query.vue'
import PeriodOfTimeQuery from './period-of-time-query.vue'
import type { FindQueryEditorViewProps } from './find-query-editor-view-props'
import type { FindQueryEditorViewEmits } from './find-query-editor-view-emits'
import { useFindQueryEditorView } from '@/classes/use-find-query-editor-view'

const props = defineProps<FindQueryEditorViewProps>()
const emits = defineEmits<FindQueryEditorViewEmits>()

const {
    // Template refs
    keyword_query,
    timeis_query,
    rep_query,
    tag_query,
    map_query,
    period_of_time_query,

    // State
    query,
    loading,
    inited_keyword_query_for_query_sidebar,
    inited_timeis_query_for_query_sidebar,
    inited_rep_query_for_query_sidebar,
    inited_tag_query_for_query_sidebar,
    inited_map_query_for_query_sidebar,
    inited_period_of_time_query_for_query_sidebar,

    // Computed
    loading_class,
    inited,

    // Exposed methods
    generate_query,
    get_default_query,

    // Template event handlers
    emits_current_query,
    emits_cleard_keyword_query,
    emits_cleard_timeis_query,
    emits_cleard_rep_query,
    emits_cleard_tag_query,
    emits_cleard_map_query,
    emits_cleard_period_of_time_query,
    emits_default_query,
    onRepQueryRequestUpdateCheckedReps,
    onTagQueryRequestUpdateCheckedTags,
    onTimeisQueryRequestUpdateCheckedTimeisTags,
    onInitedKeyword,
    onInitedTimeis,
    onInitedRep,
    onInitedTag,
    onInitedMap,
    onSaveClicked,
} = useFindQueryEditorView({ props, emits })

defineExpose({ generate_query, get_default_query })
</script>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    display: flex;
    flex-direction: column;
    align-items: center;
    min-height: 40vh;
    min-width: 40vw;
}

.find_query_editor_view.dummy {
    position: relative;
    min-height: 40vh;
    min-width: 40vw;
}

.loading_find_query_editor_view {
    overflow-y: hidden !important;
    min-width: calc(40vw + 8px);
}
</style>
