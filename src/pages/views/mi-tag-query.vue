<template>
    <FoldableStruct :application_config="application_config" :gkill_api="gkill_api" :folder_name="''" :is_open="true"
        :query="query.generate_find_kyou_query()" :struct_obj="application_config.tag_struct"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" 
        @requested_update_check_state="(items, check_state) => update_check_state(items, check_state)"
        @clicked_items="'//TODO'"
        />
</template>
<script setup lang="ts">
import type { CheckState } from './check-state'
import FoldableStruct from './foldable-struct.vue'
import type { miTagQueryEmits } from './mi-tag-query-emits'
import type { miTagQueryProps } from './mi-tag-query-props'

const props = defineProps<miTagQueryProps>()
const emits = defineEmits<miTagQueryEmits>()

async function update_check_state(items: Array<string>, check_state: CheckState) : Promise<void> {
}
</script>
