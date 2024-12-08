<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="query.use_timeis"
                    @change="emits('request_update_use_timeis_query', query.use_timeis)" label="状況" hide-details
                    class="pa-0 ma-0" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
                <v-btn @click="emits('request_clear_timeis_query')" hide-details>クリア</v-btn>
            </v-col>
        </v-row>
        <v-row v-show="query.use_timeis" class="pa-0 ma-0">
            <v-col cols="2" class="pa-0 ma-0">
                <v-btn v-if="query.timeis_words_and" icon="mdi-set-center"
                    @click="query.timeis_words_and = !query.timeis_words_and ;emits('request_update_and_search_timeis_word', query.timeis_words_and)" />
                <v-btn v-if="!query.timeis_words_and" icon="mdi-set-all"
                    @click="query.timeis_words_and = !query.timeis_words_and ;emits('request_update_and_search_timeis_word', query.timeis_words_and)" />
            </v-col>
            <v-col cols="10" class="pa-0 ma-0">
                <v-text-field v-model="query.timeis_keywords" label="状況キーワード" hide-details
                    @change="emits('request_update_timeis_keywords', query.timeis_keywords)" />
            </v-col>
        </v-row>
        <v-row v-show="query.use_timeis" class="pa-0 ma-0">
            <v-col cols="2" class="pa-0 ma-0">
                <v-btn v-if="query.use_timeis_tags" icon="mdi-set-center"
                    @click="query.use_timeis_tags = !query.use_timeis_tags; emits('request_update_and_search_timeis_tags', query.use_timeis_tags)" />
                <v-btn v-if="!query.use_timeis_tags" icon="mdi-set-all"
                    @click="query.use_timeis_tags = !query.use_timeis_tags; emits('request_update_and_search_timeis_tags', query.use_timeis_tags)" />
            </v-col>
            <v-col cols="10" class="pt-4 pa-0 ma-0">
                <v-checkbox v-model="query.use_timeis_tags"
                    @change="emits('request_update_use_timeis_query', query.use_timeis_tags)" label="状況タグ" hide-details
                    class="pa-0 ma-0" />
            </v-col>
        </v-row>
    </div>
    <table v-show="query.use_timeis_tags" class="taglist">
        <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true" :struct_obj="cloned_application_config.parsed_tag_struct" :is_editable="false"
            :is_root="true" :is_show_checkbox="true" @clicked_items="clicked_items"
            @requested_update_check_state="update_check_state"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
    </table>
</template>
<script setup lang="ts">
import type { TimeIsQueryEmits } from './time-is-query-emits'
import type { TimeIsQueryProps } from './time-is-query-props'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { type Ref, ref, watch } from 'vue'

import FoldableStruct from './foldable-struct.vue'
import { CheckState } from './check-state'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { FoldableStructModel } from './foldable-struct-model'

const props = defineProps<TimeIsQueryProps>()
const emits = defineEmits<TimeIsQueryEmits>()
defineExpose({ get_use_timeis, get_use_and_search_timeis_words, get_use_and_search_timeis_tags, get_timeis_keywords, get_use_timeis_tags, get_timeis_tags })

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)
const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

watch(() => props.application_config, async () => {
    cloned_application_config.value = props.application_config.clone()
    const errors = await cloned_application_config.value.load_all()
    if (errors !== null && errors.length !== 0) {
        emits('received_errors', errors)
        return
    }
    await update_check_state(query.value.timeis_tags, CheckState.checked)
    const checked_items = foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_timeis_tags', checked_items, false)
    }
})

watch(() => props.find_kyou_query, async () => {
    query.value = props.find_kyou_query.clone()
    await update_check_state(query.value.timeis_tags, CheckState.checked)
    const checked_items = foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_timeis_tags', checked_items, false)
        emits('inited')
    }
})

const query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())

async function clicked_items(e: MouseEvent, items: Array<string>, check_state: CheckState, is_user: boolean): Promise<void> {
    const checked_items = foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_timeis_tags', checked_items, true)
    }
}

function get_use_timeis(): boolean {
    return query.value.use_timeis
}
function get_use_timeis_tags(): boolean {
    return query.value.use_timeis_tags
}
function get_use_and_search_timeis_words(): boolean {
    return query.value.timeis_words_and
}
function get_use_and_search_timeis_tags(): boolean {
    return query.value.timeis_tags_and
}
function get_timeis_keywords(): string {
    return query.value.timeis_keywords
}
function get_timeis_tags(): Array<string> {
    const tags = foldable_struct.value?.get_selected_items()
    if (tags) {
        return tags
    }
    return new Array<string>()
}

async function update_check_state(items: Array<string>, is_checked: CheckState): Promise<void> {
    update_check(items, is_checked, false)
}

async function update_check(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean): Promise<void> {
    if (pre_uncheck_all) {
        let f = (struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            struct.is_checked = false
            struct.indeterminate = false
            if (struct.children) {
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        f(cloned_application_config.value.parsed_tag_struct)
    }

    for (let i = 0; i < items.length; i++) {
        const key_name = items[i]
        let f = (struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            if (struct.key === key_name) {
                switch (is_checked) {
                    case CheckState.checked:
                        struct.is_checked = true
                        struct.indeterminate = false
                        break
                    case CheckState.unchecked:
                        struct.is_checked = false
                        struct.indeterminate = false
                        break
                    case CheckState.indeterminate:
                        struct.is_checked = false
                        struct.indeterminate = true
                        break
                }
            }
            if (struct.children) {
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        f(cloned_application_config.value.parsed_tag_struct)
    }

    const checked_items = foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_timeis_tags', checked_items, true)
    }
}


</script>
