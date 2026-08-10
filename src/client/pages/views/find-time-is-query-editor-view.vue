<template>
    <v-card variant="flat" class="find_time_is_query_editor_view dummy" :class="loading_class">
        <div class="overlay_target">
            <v-overlay v-model="loading" class="align-center justify-center" persistent contained>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
        </div>
        <div v-show="inited" class="find_time_is_query_editor_view">
            <KeywordQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search="emits_current_query()" @request_update_keywords="emits_current_query()"
                @request_update_use_keyword_query="emits_current_query()"
                @request_clear_keyword_query="emits_cleard_keyword_query()"
                :inited="inited_keyword_query_for_query_sidebar" @inited="onInitedKeyword"
                ref="keyword_query" />
            <div> <v-divider /> </div>
            <v-checkbox v-model="use_tag_filter" :label="i18n.global.t('USE_TAG_FILTER_TITLE')" hide-details
                density="compact" class="pl-2" />
            <TagQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_tags="emits_current_query()"
                @request_update_checked_tags="(_tags, is_by_user) => onTagQueryRequestUpdateCheckedTags(_tags, is_by_user)"
                @request_clear_tag_query="emits_cleard_tag_query()" ref="tag_query"
                :inited="inited_tag_query_for_query_sidebar" @inited="onInitedTag" />
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
import type { FindTimeIsQueryEditorViewEmits } from './find-time-is-query-editor-view-emits'
import type { FindTimeIsQueryEditorViewProps } from './find-time-is-query-editor-view-props'
import { useFindTimeIsQueryEditorView } from '@/classes/use-find-time-is-query-editor-view'

const props = defineProps<FindTimeIsQueryEditorViewProps>()
const emits = defineEmits<FindTimeIsQueryEditorViewEmits>()

const {
    // Template refs
    keyword_query,
    tag_query,

    // State
    query,
    use_tag_filter,
    loading,
    inited_keyword_query_for_query_sidebar,
    inited_tag_query_for_query_sidebar,

    // Computed
    loading_class,
    inited,

    // Exposed methods
    generate_query,
    get_default_query,

    // Template event handlers
    emits_current_query,
    emits_cleard_keyword_query,
    emits_cleard_tag_query,
    emits_default_query,
    onTagQueryRequestUpdateCheckedTags,
    onInitedKeyword,
    onInitedTag,
    onSaveClicked,
} = useFindTimeIsQueryEditorView({ props, emits })

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
}

/* min-width を持たせない理由は find-query-editor-view.vue と同じ（横スクロールの二重化を防ぐ） */
.find_time_is_query_editor_view.dummy {
    position: relative;
    min-height: 40vh;
}

.loading_find_time_is_query_editor_view {
    overflow-y: hidden !important;
}
</style>
