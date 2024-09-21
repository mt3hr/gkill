<template>
    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api" :is_open="true"
        :query="query" :struct_obj="application_config.rep_type_struct" @requested_update_check_state="'//TODO'"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" @clicked_items="'//TODO'" />
    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api" :is_open="true"
        :query="query" :struct_obj="application_config.rep_struct" @requested_update_check_state="'//TODO'"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" @clicked_items="'//TODO'" />
    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api" :is_open="true"
        :query="query" :struct_obj="application_config.device_struct" @requested_update_check_state="'//TODO'"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" @clicked_items="'//TODO'" />
</template>
<script setup lang="ts">
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { type Ref, ref } from 'vue'
import FoldableStruct from './foldable-struct.vue'
import type { RepQueryEmits } from './rep-query-emits'
import type { RepQueryProps } from './rep-query-props'

const props = defineProps<RepQueryProps>()
const emits = defineEmits<RepQueryEmits>()
const cloned_query: Ref<FindKyouQuery> = ref(await props.query.clone())
</script>
<style scoped></style>
