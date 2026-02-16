<template>
    <v-dialog persistent @click:outside="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <KFTLTemplateView :application_config="application_config" :gkill_api="gkill_api" :template="template"
            @clicked_template_element_leaf="(template: KFTLTemplateElementData) => emits('clicked_template_element_leaf', template)"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide()" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import type { KFTLTemplateDialogEmits } from './kftl-template-dialog-emits'
import type { KFTLTemplateDialogProps } from './kftl-template-dialog-props'
import KFTLTemplateView from '../views/kftl-template-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<KFTLTemplateDialogProps>()
const emits = defineEmits<KFTLTemplateDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    emits('closed_dialog')
}
</script>
