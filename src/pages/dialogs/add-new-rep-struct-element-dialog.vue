<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <AddNewRepStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide"
            @requested_add_rep_struct_element="(...rep_struct_element :any[]) => emits('requested_add_rep_struct_element', rep_struct_element[0] as RepStructElementData)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewRepStructElementDialogEmits } from './add-new-rep-struct-element-dialog-emits'
import type { AddNewRepStructElementDialogProps } from './add-new-rep-struct-element-dialog-props'
import AddNewRepStructElementView from '../views/add-new-rep-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'

const add_new_rep_struct_element_view = ref<InstanceType<typeof AddNewRepStructElementView> | null>(null);

defineProps<AddNewRepStructElementDialogProps>()
const emits = defineEmits<AddNewRepStructElementDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    add_new_rep_struct_element_view.value?.reset_rep_name()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    add_new_rep_struct_element_view.value?.reset_rep_name()
}
</script>
