<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="cloned_query.use_timeis"
                    @change="emits('request_update_use_timeis_query', cloned_query.use_timeis)"
                    :label="i18n.global.t('PLAING_TIMEIS_QUERY_TITLE')" hide-details class="pa-0 ma-0" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
                <v-btn dark color="secondary" @click="emits('request_clear_timeis_query')" hide-details>{{
                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row v-show="cloned_query.use_timeis" class="pa-0 ma-0">
            <v-col cols="2" class="pa-0 ma-0">
                <v-btn v-if="cloned_query.timeis_words_and" icon="mdi-set-center"
                    @click="cloned_query.timeis_words_and = !cloned_query.timeis_words_and; emits('request_update_and_search_timeis_word', cloned_query.timeis_words_and)" />
                <v-btn v-if="!cloned_query.timeis_words_and" icon="mdi-set-all"
                    @click="cloned_query.timeis_words_and = !cloned_query.timeis_words_and; emits('request_update_and_search_timeis_word', cloned_query.timeis_words_and)" />
            </v-col>
            <v-col cols="10" class="pa-0 ma-0">
                <v-text-field v-model="cloned_query.timeis_keywords"
                    :label="i18n.global.t('PLAING_TIMEIS_QUERY_KEYWORD_TITLE')" hide-details
                    @change="emits('request_update_timeis_keywords', cloned_query.timeis_keywords)" />
            </v-col>
        </v-row>
        <v-row v-show="cloned_query.use_timeis" class="pa-0 ma-0">
            <v-col cols="2" class="pa-0 ma-0">
                <v-btn v-if="cloned_query.timeis_tags_and" icon="mdi-set-center"
                    @click="cloned_query.timeis_tags_and = !cloned_query.timeis_tags_and; emits('request_update_and_search_timeis_tags', cloned_query.timeis_tags_and)" />
                <v-btn v-if="!cloned_query.timeis_tags_and" icon="mdi-set-all"
                    @click="cloned_query.timeis_tags_and = !cloned_query.timeis_tags_and; emits('request_update_and_search_timeis_tags', cloned_query.timeis_tags_and)" />
            </v-col>
            <v-col cols="10" class="pa-0 ma-0">
                <v-checkbox v-model="cloned_query.use_timeis_tags"
                    @click="cloned_query.use_timeis_tags = !cloned_query.use_timeis_tags; emits('request_update_use_timeis_query', cloned_query.use_timeis_tags)"
                    :label="i18n.global.t('PLAING_TIMEIS_TAG_TITLE')" hide-details class="pa-0 ma-0" />
            </v-col>
        </v-row>
    </div>
    <table v-show="cloned_query.use_timeis_tags" class="taglist">
        <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true" :struct_obj="cloned_application_config.parsed_tag_struct" :is_editable="false"
            :is_root="true" :is_show_checkbox="true" @clicked_items="clicked_items"
            @requested_update_check_state="update_check_state"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
    </table>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { TimeIsQueryEmits } from './time-is-query-emits'
import type { TimeIsQueryProps } from './time-is-query-props'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { nextTick, type Ref, ref, watch } from 'vue'

import FoldableStruct from './foldable-struct.vue'
import { CheckState } from './check-state'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { FoldableStructModel } from './foldable-struct-model'

const props = defineProps<TimeIsQueryProps>()
const emits = defineEmits<TimeIsQueryEmits>()
defineExpose({ get_use_timeis, get_use_and_search_timeis_words, get_use_and_search_timeis_tags, get_timeis_keywords, get_use_timeis_tags, get_timeis_tags, update_check })

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)
const old_cloned_query: Ref<FindKyouQuery | null> = ref(null)
const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
const cloned_query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())

const loading = ref(false)
watch(() => loading.value, async (new_value: boolean, old_value: boolean) => {
    if (new_value !== old_value && new_value) {
        const tags = cloned_query.value.tags
        if (tags) {
            await update_check(tags, CheckState.checked, true)
        }
    }
})

const skip_emits_this_tick = ref(false)
watch(() => props.application_config, async () => {
    loading.value = true
    cloned_query.value = props.find_kyou_query
    cloned_application_config.value = props.application_config.clone()
    if (props.inited) {
        skip_emits_this_tick.value = true
        nextTick(() => skip_emits_this_tick.value = false)
        update_check(cloned_query.value.timeis_tags, CheckState.checked, true)
        return
    }
    const tags = Array<string>()
    cloned_application_config.value.tag_struct.forEach(tag => {
        if (tag.check_when_inited) {
            tags.push(tag.tag_name)
        }
    })
    if (!props.inited) {
        emits('inited')
    }
})

watch(() => props.find_kyou_query, async (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
    loading.value = true
    old_cloned_query.value = old_value
    cloned_query.value = new_value.clone()
    cloned_query.value = props.find_kyou_query.clone()
    await update_check_state(cloned_query.value.timeis_tags, CheckState.checked)
    const checked_items = foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_timeis_tags', checked_items, false)
    }
})

async function clicked_items(_e: MouseEvent, items: Array<string>, check_state: CheckState): Promise<void> {
    update_check(items, check_state, true)
}

function get_use_timeis(): boolean {
    return cloned_query.value.use_timeis
}
function get_use_timeis_tags(): boolean {
    return cloned_query.value.use_timeis_tags
}
function get_use_and_search_timeis_words(): boolean {
    return cloned_query.value.timeis_words_and
}
function get_use_and_search_timeis_tags(): boolean {
    return cloned_query.value.timeis_tags_and
}
function get_timeis_keywords(): string {
    return cloned_query.value.timeis_keywords
}
function get_timeis_tags(): Array<string> {
    const tags = foldable_struct.value?.get_selected_items()
    if (tags) {
        return tags
    }
    return new Array<string>()
}

async function update_check_state(items: Array<string>, is_checked: CheckState): Promise<void> {
    await update_check(items, is_checked, false)
}

async function update_check(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean, disable_emits?: boolean): Promise<void> {
    if (pre_uncheck_all) {
        let f = (_struct: FoldableStructModel) => { }
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
        let f = (_struct: FoldableStructModel) => { }
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
        if (!skip_emits_this_tick.value && !disable_emits) {
            emits('request_update_checked_timeis_tags', checked_items, true)
        }
    }
    foldable_struct.value?.update_check()
}
</script>
