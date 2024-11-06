<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <v-btn v-if="!is_and_search" icon="mdi-set-center"
                @click="is_and_search = !is_and_search; emits('request_update_and_search_tags', is_and_search)" />
            <v-btn v-if="is_and_search" icon="mdi-set-all"
                @click="is_and_search = !is_and_search; emits('request_update_and_search_tags', is_and_search)" />
        </v-col>
        <v-col cols="auto" class="pa-0 ma-0">
            <v-checkbox readonly v-model="use_tag" label="タグ" />
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pb-0 mb-0 pr-0">
            <v-btn @click="emits('request_clear_tag_query')" hide-details>クリア</v-btn>
        </v-col>
    </v-row>
    <table class="taglist">
        <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true" :query="query" :struct_obj="cloned_application_config.parsed_tag_struct"
            @clicked_items="clicked_items" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
    </table>
</template>
<script setup lang="ts">
import { readonly, type Ref, ref } from 'vue'
import type { TagQueryEmits } from './tag-query-emits'
import type { TagQueryProps } from './tag-query-props'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import FoldableStruct from './foldable-struct.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { CheckState } from './check-state'

const props = defineProps<TagQueryProps>()
const emits = defineEmits<TagQueryEmits>()


const use_tag: Ref<boolean> = ref(true)
const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)
const is_and_search: Ref<boolean> = ref(false)

const cloned_query: Ref<FindKyouQuery> = ref(await props.query.clone())
const cloned_application_config: Ref<ApplicationConfig> = ref(await props.application_config.clone())

async function clicked_items(items: Array<string>, is_checked: CheckState): Promise<void> {
    const checked_items = await foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_tags', checked_items)
    }
}
</script>
