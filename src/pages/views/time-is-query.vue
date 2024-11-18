<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="use_timeis" @click="emits('request_update_use_timeis_query', use_timeis)"
                    label="状況" hide-details class="pa-0 ma-0" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
                <v-btn @click="emits('request_clear_timeis_query')" hide-details>クリア</v-btn>
            </v-col>
        </v-row>
        <v-row v-show="use_timeis" class="pa-0 ma-0">
            <v-col cols="2" class="pa-0 ma-0">
                <v-btn v-if="use_and_search_timeis_words" icon="mdi-set-center"
                    @click="use_and_search_timeis_words = !use_and_search_timeis_words; emits('request_update_and_search_timeis_word', use_and_search_timeis_tags)" />
                <v-btn v-if="!use_and_search_timeis_words" icon="mdi-set-all"
                    @click="use_and_search_timeis_words = !use_and_search_timeis_words; emits('request_update_and_search_timeis_word', use_and_search_timeis_tags)" />
            </v-col>
            <v-col cols="10" class="pa-0 ma-0">
                <v-text-field v-model="timeis_keywords" label="状況キーワード" hide-details />
            </v-col>
        </v-row>
        <v-row v-show="use_timeis" class="pa-0 ma-0">
            <v-col cols="2" class="pa-0 ma-0">
                <v-btn v-if="use_and_search_timeis_tags" icon="mdi-set-center"
                    @click="use_and_search_timeis_tags = !use_and_search_timeis_tags; emits('request_update_and_search_timeis_tags', use_and_search_timeis_tags)" />
                <v-btn v-if="!use_and_search_timeis_tags" icon="mdi-set-all"
                    @click="use_and_search_timeis_tags = !use_and_search_timeis_tags; emits('request_update_and_search_timeis_tags', use_and_search_timeis_tags)" />
            </v-col>
            <v-col cols="10" class="pt-4 pa-0 ma-0">
                <v-label>状況タグ</v-label>
            </v-col>
        </v-row>
        <table v-show="use_timeis" class="taglist">
            <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                :is_open="false" :query="query" :struct_obj="cloned_application_config.parsed_tag_struct" @clicked_items="clicked_items"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
        </table>
    </div>
</template>
<script setup lang="ts">
import type { TimeIsQueryEmits } from './time-is-query-emits'
import type { TimeIsQueryProps } from './time-is-query-props'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { type Ref, ref } from 'vue'

import FoldableStruct from './foldable-struct.vue'
import type { CheckState } from './check-state'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'

const props = defineProps<TimeIsQueryProps>()
const emits = defineEmits<TimeIsQueryEmits>()

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)
const use_timeis: Ref<boolean> = ref(false)
const use_and_search_timeis_words: Ref<boolean> = ref(false)
const use_and_search_timeis_tags: Ref<boolean> = ref(false)
const timeis_keywords: Ref<string> = ref(props.query.timeis_word.join(" ") + props.query.timeis_not_word.join(" -"))
const cloned_application_config: Ref<ApplicationConfig> = ref(await props.application_config.clone())

const query: Ref<FindKyouQuery> = ref(await props.query.clone())

async function clicked_items(e: MouseEvent, items: Array<string>, check_state: CheckState, is_user: boolean): Promise<void> {
    const checked_items = await foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_timeis_tags', checked_items)
    }
}
</script>
