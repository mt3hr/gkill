<template>
    <!-- //TODO クリアするやつ -->
    <!-- //TODO AND ORのやつ-->
    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api" :is_open="true"
        :query="query" :struct_obj="cloned_application_config.parsed_tag_struct" @clicked_items="clicked_items"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { TagQueryEmits } from './tag-query-emits'
import type { TagQueryProps } from './tag-query-props'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import FoldableStruct from './foldable-struct.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

const props = defineProps<TagQueryProps>()
const emits = defineEmits<TagQueryEmits>()

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)

const cloned_query: Ref<FindKyouQuery> = ref(await props.query.clone())
const cloned_application_config: Ref<ApplicationConfig> = ref(await props.application_config.clone())

async function clicked_items(items: Array<string>, is_checked: boolean): Promise<void> {
    const checked_items = await foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_tags', checked_items)
    }
}
</script>
