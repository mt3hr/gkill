<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <EditKFTLTemplateStructView :application_config="application_config" :gkill_api="gkill_api"
            :kftl_template_struct="application_config.kftl_template_struct"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_application_config="emits('requested_reload_application_config')"
            @requested_apply_kftl_template_struct="(...kftl_template_struct_element_data: any[]) => emits('requested_apply_kftl_template_struct', kftl_template_struct_element_data[0] as KFTLTemplateStructElementData)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditKFTLTemplateStructDialogEmits } from './edit-kftl-template-struct-dialog-emits.ts'
import type { EditKFTLTemplateStructDialogProps } from './edit-kftl-template-struct-dialog-props.ts'
import EditKFTLTemplateStructView from '../views/edit-kftl-template-struct-view.vue'
import type { GkillError } from '@/classes/api/gkill-error.js'
import type { GkillMessage } from '@/classes/api/gkill-message.js'

defineProps<EditKFTLTemplateStructDialogProps>()
const emits = defineEmits<EditKFTLTemplateStructDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import type { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data.js'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}

</script>
