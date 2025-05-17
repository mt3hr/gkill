<template>
    <div>
        <v-checkbox readonly v-model="use_board" :label="i18n.global.t('BOARD_TITLE')" hide-details />
        <table v-show="use_board" class="boardlist">
            <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                :is_open="true" :struct_obj="cloned_application_config.parsed_mi_boad_struct" :is_editable="false"
                :is_root="true" :is_show_checkbox="false"
                @clicked_items="(event: MouseEvent, items: string[], check_state: CheckState, is_by_user: boolean) => { if (is_by_user && check_state === CheckState.checked) { items.forEach((board) => { board_name = board; emits('request_open_focus_board', board) }) } }"
                @requested_update_check_state="[]" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
        </table>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { miBoardQueryEmits } from './mi-board-query-emits'
import type { miBoardQueryProps } from './mi-board-query-props'
import { nextTick, type Ref, ref, watch } from 'vue'
import FoldableStruct from './foldable-struct.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { CheckState } from './check-state'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)

const props = defineProps<miBoardQueryProps>()
const emits = defineEmits<miBoardQueryEmits>()
defineExpose({ get_board_name })

const cloned_query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())
const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
const board_name: Ref<string> = ref(i18n.global.t("MI_ALL_TITLE"))

const use_board = ref(true)

const skip_emits_this_tick = ref(false)
watch(() => props.application_config, async () => {
    cloned_application_config.value = props.application_config.clone()
    const errors = await cloned_application_config.value.load_all()
    if (errors !== null && errors.length !== 0) {
        emits('received_errors', errors)
        return
    }
    if (props.inited) {
        skip_emits_this_tick.value = true
        nextTick(() => skip_emits_this_tick.value = false)
        return
    }
    emits('inited')
})

watch(() => props.find_kyou_query, async () => {
    cloned_query.value = props.find_kyou_query.clone()
    board_name.value = cloned_query.value.mi_board_name
})

function get_board_name(): string {
    return board_name.value
}
</script>
