<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox readonly v-model="use_tag" :label="i18n.global.t('TAG_QUERY_TITLE')" hide-details />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
                <v-btn dark color="secondary" @click="emits('request_clear_tag_query')" hide-details>{{
                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn v-if="is_and_search" icon="mdi-set-center"
                    @click="is_and_search = !is_and_search; emits('request_update_and_search_tags', is_and_search)" />
                <v-btn v-if="!is_and_search" icon="mdi-set-all"
                    @click="is_and_search = !is_and_search; emits('request_update_and_search_tags', is_and_search)" />
            </v-col>
        </v-row>
        <table v-show="use_tag" class="taglist">
            <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                :is_open="true" :struct_obj="cloned_application_config.parsed_tag_struct" :is_editable="false"
                :is_root="true" :is_show_checkbox="true" @clicked_items="clicked_items"
                @requested_update_check_state="update_check_state"
                @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" ref="foldable_struct" />
        </table>
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref, watch } from 'vue'
import type { TagQueryEmits } from './tag-query-emits'
import type { TagQueryProps } from './tag-query-props'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import FoldableStruct from './foldable-struct.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { CheckState } from './check-state'
import type { FoldableStructModel } from './foldable-struct-model'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const props = defineProps<TagQueryProps>()
const emits = defineEmits<TagQueryEmits>()
defineExpose({ get_use_tag, get_tags, get_is_and_search, update_check })

const use_tag: Ref<boolean> = ref(true)
const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)
const is_and_search: Ref<boolean> = ref(false)

const old_cloned_query: Ref<FindKyouQuery | null> = ref(null)
const cloned_query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())
const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

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
    cloned_application_config.value = props.application_config.clone()
    if (props.inited) {
        skip_emits_this_tick.value = true
        nextTick(() => skip_emits_this_tick.value = false)
        update_check(cloned_query.value.tags, CheckState.checked, true)
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
    is_and_search.value = props.find_kyou_query.tags_and
    await update_check_state(cloned_query.value.tags, CheckState.checked)
    const checked_items = foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_tags', checked_items, false)
    }
})

async function clicked_items(e: MouseEvent, items: Array<string>, is_checked: CheckState): Promise<void> {
    update_check(items, is_checked, true)
}

function update_check_state(items: Array<string>, is_checked: CheckState) {
    update_check(items, is_checked, false)
}

function update_check(items: Array<string>, is_checked: CheckState, pre_uncheck_all: boolean, disable_emits?: boolean){
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
            emits('request_update_checked_tags', checked_items, true)
        }
    }
    foldable_struct.value?.update_check()
}

function get_use_tag(): boolean {
    return use_tag.value
}
function get_tags(): Array<string> | null {
    const tags = foldable_struct.value?.get_selected_items()
    if (!tags) {
        return null
    }
    return tags
}
function get_is_and_search(): boolean {
    return is_and_search.value
}
</script>
