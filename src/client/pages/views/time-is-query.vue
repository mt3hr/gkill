<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="cloned_query.use_timeis"
                    @change="onChangeUseTimeis"
                    :label="i18n.global.t('PLAING_TIMEIS_QUERY_TITLE')" hide-details class="pa-0 ma-0" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
                <v-btn dark color="secondary" @click="onClickClear" hide-details>{{
                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row v-show="cloned_query.use_timeis" class="pa-0 ma-0">
            <v-col cols="2" class="pa-0 ma-0">
                <v-btn v-if="cloned_query.timeis_words_and" icon="mdi-set-center"
                    @click="onToggleTimeisWordsAnd" />
                <v-btn v-if="!cloned_query.timeis_words_and" icon="mdi-set-all"
                    @click="onToggleTimeisWordsAnd" />
            </v-col>
            <v-col cols="10" class="pa-0 ma-0">
                <v-text-field v-model="cloned_query.timeis_keywords"
                    :label="i18n.global.t('PLAING_TIMEIS_QUERY_KEYWORD_TITLE')" hide-details
                    @change="onChangeTimeisKeywords" />
            </v-col>
        </v-row>
        <v-row v-show="cloned_query.use_timeis" class="pa-0 ma-0">
            <v-col cols="2" class="pa-0 ma-0">
                <v-btn v-if="cloned_query.timeis_tags_and" icon="mdi-set-center"
                    @click="onToggleTimeisTagsAnd" />
                <v-btn v-if="!cloned_query.timeis_tags_and" icon="mdi-set-all"
                    @click="onToggleTimeisTagsAnd" />
            </v-col>
            <v-col cols="10" class="pa-0 ma-0">
                <v-checkbox v-model="cloned_query.use_timeis_tags"
                    @click="onClickUseTimeisTags"
                    :label="i18n.global.t('PLAING_TIMEIS_TAG_TITLE')" hide-details class="pa-0 ma-0" />
            </v-col>
        </v-row>
    </div>
    <table v-show="cloned_query.use_timeis && cloned_query.use_timeis_tags" class="taglist">
        <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true" :struct_obj="cloned_application_config.tag_struct" :is_editable="false"
            :is_root="true" :is_show_checkbox="true" @clicked_items="clicked_items"
            @requested_update_check_state="update_check_state"
            @received_errors="foldableStructHandlers['received_errors']"
            @received_messages="foldableStructHandlers['received_messages']"
            ref="foldable_struct" />
    </table>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { TimeIsQueryEmits } from './time-is-query-emits'
import type { TimeIsQueryProps } from './time-is-query-props'

import FoldableStruct from './foldable-struct.vue'
import { useTimeIsQuery } from '@/classes/use-time-is-query'

const props = defineProps<TimeIsQueryProps>()
const emits = defineEmits<TimeIsQueryEmits>()

const {
    foldable_struct,
    cloned_application_config,
    cloned_query,
    get_use_timeis,
    get_use_and_search_timeis_words,
    get_use_and_search_timeis_tags,
    get_timeis_keywords,
    get_use_timeis_tags,
    get_timeis_tags,
    update_check,
    clicked_items,
    update_check_state,
    onChangeUseTimeis,
    onClickClear,
    onToggleTimeisWordsAnd,
    onChangeTimeisKeywords,
    onToggleTimeisTagsAnd,
    onClickUseTimeisTags,
    foldableStructHandlers,
} = useTimeIsQuery({ props, emits })

defineExpose({ get_use_timeis, get_use_and_search_timeis_words, get_use_and_search_timeis_tags, get_timeis_keywords, get_use_timeis_tags, get_timeis_tags, update_check })
</script>
