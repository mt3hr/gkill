<template>
    <v-card variant="flat" class="mi_find_query_editor_view dummy" :class="loading_class">
        <div class="overlay_target">
            <v-overlay v-model="loading" class="align-center justify-center" persistent contained>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
        </div>
        <div v-show="inited" class="mi_find_query_editor_view">
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
                @request_clear_sort_type="emits_cleard_sort_type_query()"
                @inited="onInitedSort" ref="sort_type_query" />
            <div> <v-divider /> </div>
            <TagQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_tags="emits_current_query()"
                @request_update_checked_tags="(_tags, is_by_user) => onTagQueryRequestUpdateCheckedTags(_tags, is_by_user)"
                @request_clear_tag_query="emits_cleard_tag_query()"
                ref="tag_query" :inited="inited_tag_query_for_query_sidebar"
                @inited="onInitedTag" />
        </div>
        <v-card-action v-if="!loading">
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="onSaveClicked">{{
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
import TagQuery from './tag-query.vue'
import miExtractCheckStateQuery from './mi-extract-check-state-query.vue'
import miSortTypeQuery from './mi-sort-type-query.vue'
import type { MiFindQueryEditorViewEmits } from './mi-find-query-editor-view-emits'
import type { MiFindQueryEditorViewProps } from './mi-find-query-editor-view-props'
import { useMiFindQueryEditorView } from '@/classes/use-mi-find-query-editor-view'

const props = defineProps<MiFindQueryEditorViewProps>()
const emits = defineEmits<MiFindQueryEditorViewEmits>()

const {
    // Template refs
    keyword_query,
    tag_query,
    check_state_query,
    sort_type_query,

    // State
    query,
    loading,
    inited_keyword_query_for_query_sidebar,
    inited_tag_query_for_query_sidebar,
    inited_check_state_query_for_query_sidebar: _inited_check_state_query_for_query_sidebar,
    inited_sort_query_for_query_sidebar: _inited_sort_query_for_query_sidebar,

    // Computed
    loading_class,
    inited,

    // Exposed methods
    generate_query,
    get_default_query,

    // Template event handlers
    emits_current_query,
    emits_cleard_check_state,
    emits_cleard_sort_type_query,
    emits_cleard_keyword_query,
    emits_cleard_tag_query,
    emits_default_query,
    onTagQueryRequestUpdateCheckedTags,
    onInitedKeyword,
    onInitedTag,
    onInitedCheckState,
    onInitedSort,
    onSaveClicked,
} = useMiFindQueryEditorView({ props, emits })

defineExpose({ generate_query, get_default_query })
</script>
