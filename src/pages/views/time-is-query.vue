<template>
    <!-- //TODO クリアするやつ -->
    <!-- //TODO AND ORのやつ-->
    <!-- //TODO WORDのやつ-->
    <!-- //TODO USEするかどうかのやつ-->
    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api" :is_open="true"
        :query="query" :struct_obj="application_config.parsed_tag_struct" @clicked_items="clicked_items"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
    <!--//TODO-->
    <!--//TODO 実装-->
</template>
<script setup lang="ts">
import type { TimeIsQueryEmits } from './time-is-query-emits'
import type { TimeIsQueryProps } from './time-is-query-props'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { type Ref, ref } from 'vue'

import FoldableStruct from './foldable-struct.vue'

const props = defineProps<TimeIsQueryProps>()
const emits = defineEmits<TimeIsQueryEmits>()

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)

const query: Ref<FindKyouQuery> = ref(await props.query.clone())

async function clicked_items(items: Array<string>, is_checked: boolean): Promise<void> {
    const checked_items = await foldable_struct.value?.get_selected_items()
    if (checked_items) {
        emits('request_update_checked_timeis_tags', checked_items)
    }
}
</script>
