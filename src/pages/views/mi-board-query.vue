<template>
    <div>
        <v-checkbox readonly v-model="use_board" :label="i18n.global.t('BOARD_TITLE')" hide-details />
        <table v-show="use_board" class="boardlist">
            <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                :is_open="true" :struct_obj="mi_board_struct" :is_editable="false" :is_root="true"
                :is_show_checkbox="false"
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
import { computed, nextTick, type Ref, ref } from 'vue'
import FoldableStruct from './foldable-struct.vue'
import { CheckState } from './check-state'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)

const props = defineProps<miBoardQueryProps>()
const emits = defineEmits<miBoardQueryEmits>()
defineExpose({ get_board_name })
const mi_board_struct = computed(() => props.application_config.mi_board_struct)

const board_name: Ref<string> = ref(i18n.global.t("MI_ALL_TITLE"))
nextTick(() => emits('inited'))

const use_board = ref(true)
function get_board_name(): string {
    return board_name.value
}
</script>
