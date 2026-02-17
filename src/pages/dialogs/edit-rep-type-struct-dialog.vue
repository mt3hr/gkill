<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <EditRepTypeStructView :application_config="application_config" :gkill_api="gkill_api"
            :rep_type_struct="application_config.rep_type_struct"
            @requested_reload_application_config="(...application_config: any[]) => emits('requested_reload_application_config', application_config[0] as ApplicationConfig)"
            @requested_apply_rep_type_struct="(...rep_type_struct_element_data: any[]) => emits('requested_apply_rep_type_struct', rep_type_struct_element_data[0] as RepTypeStructElementData)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditRepTypeDialogEmits } from './edit-rep-type-dialog-emits'
import type { EditRepTypeDialogProps } from './edit-rep-type-dialog-props'
import EditRepTypeStructView from '../views/edit-rep-type-struct-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

defineProps<EditRepTypeDialogProps>()
const emits = defineEmits<EditRepTypeDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import type { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
