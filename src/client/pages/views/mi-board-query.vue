<template>
    <div>
        <v-checkbox readonly v-model="use_board" :label="i18n.global.t('BOARD_TITLE')" hide-details />
        <table v-show="use_board" class="boardlist">
            <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                :is_open="true" :struct_obj="mi_board_struct" :is_editable="false" :is_root="true"
                :is_show_checkbox="false"
                @clicked_items="onClickedItems"
                @requested_update_check_state="[]"
                @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                ref="foldable_struct" />
        </table>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { MiBoardQueryEmits } from './mi-board-query-emits'
import type { MiBoardQueryProps } from './mi-board-query-props'
import { ref } from 'vue'
import FoldableStruct from './foldable-struct.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useMiBoardQuery } from '@/classes/use-mi-board-query'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null)

const props = defineProps<MiBoardQueryProps>()
const emits = defineEmits<MiBoardQueryEmits>()

const {
    mi_board_struct,
    use_board,
    get_board_name,
    onClickedItems,
} = useMiBoardQuery({ props, emits })

defineExpose({ get_board_name })
</script>
