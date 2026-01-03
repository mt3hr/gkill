<template>
    <div>
        <v-checkbox readonly v-model="use_board" :label="i18n.global.t('BOARD_TITLE')" hide-details />
        <table v-show="use_board" class="boardlist">
            <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                :is_open="true" :struct_obj="cloned_application_config.mi_board_struct" :is_editable="false"
                :is_root="true" :is_show_checkbox="false"
                @clicked_items="(event: MouseEvent, items: string[], check_state: CheckState, is_by_user: boolean) => { if (is_by_user && check_state === CheckState.checked) { items.forEach((board) => { board_name = board; emits('request_open_focus_board', board) }) } }"
                @requested_update_check_state="[]"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                ref="foldable_struct" />
        </table>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { miBoardQueryEmits } from './mi-board-query-emits'
import type { miBoardQueryProps } from './mi-board-query-props'
import { nextTick, type Ref, ref, watch } from 'vue'
import FoldableStruct from './foldable-struct.vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { CheckState } from './check-state'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)

const props = defineProps<miBoardQueryProps>()
const emits = defineEmits<miBoardQueryEmits>()
defineExpose({ get_board_name })

const cloned_query: Ref<FindKyouQuery> = ref(props.find_kyou_query.clone())
const cloned_application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
const board_name: Ref<string> = ref(i18n.global.t("MI_ALL_TITLE"))

const use_board = ref(true)

const skip_emits_this_tick = ref(false)
watch(() => props.application_config, async () => {
    cloned_application_config.value = props.application_config.clone()
    if (props.inited) {
        skip_emits_this_tick.value = true
        nextTick(() => skip_emits_this_tick.value = false)
        return
    }
    emits('inited')
})

watch(() => props.find_kyou_query, async () => {
    if (!props.find_kyou_query) {
        return
    }
    cloned_query.value = props.find_kyou_query.clone()
    board_name.value = cloned_query.value.mi_board_name
})

function get_board_name(): string {
    return board_name.value
}
</script>
