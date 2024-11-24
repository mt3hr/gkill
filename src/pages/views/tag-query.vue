<template>
    <div>
        <v-checkbox readonly v-model="use_tag" label="タグ" hide-details />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn v-if="!is_and_search" icon="mdi-set-center"
                    @click="is_and_search = !is_and_search; emits('request_update_and_search_tags', is_and_search)" />
                <v-btn v-if="is_and_search" icon="mdi-set-all"
                    @click="is_and_search = !is_and_search; emits('request_update_and_search_tags', is_and_search)" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
                <v-btn @click="emits('request_clear_tag_query')" hide-details>クリア</v-btn>
            </v-col>
        </v-row>
        <table v-show="use_tag" class="taglist">
            <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                :is_open="true" :query="query" :struct_obj="cloned_application_config.parsed_tag_struct"
                :is_editable="false" :is_root="true" :is_show_checkbox="true" @clicked_items="clicked_items"
                @requested_update_check_state="update_check_state"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
        </table>
    </div>
</template>
<script setup lang="ts">
import { readonly, type Ref, ref, watch } from 'vue'
import type { TagQueryEmits } from './tag-query-emits'
import type { TagQueryProps } from './tag-query-props'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import FoldableStruct from './foldable-struct.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { CheckState } from './check-state'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import type { FoldableStructModel } from './foldable-struct-model'
import type { RefSymbol } from '@vue/reactivity'

const props = defineProps<TagQueryProps>()
const emits = defineEmits<TagQueryEmits>()
const use_tag: Ref<boolean> = ref(true)
const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)
const is_and_search: Ref<boolean> = ref(false)

const cloned_query: Ref<FindKyouQuery> = ref(await props.query.clone())
const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

cloned_application_config.value.parse_tag_struct()

watch(() => props.application_config, async () => {
    cloned_application_config.value = props.application_config.clone()
    cloned_application_config.value.parse_tag_struct()
})

async function clicked_items(e: MouseEvent, items: Array<string>, is_checked: CheckState): Promise<void> {
    update_check(items, is_checked, true)
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

    const checked_items = await foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_tags', checked_items)
    }
}

</script>
