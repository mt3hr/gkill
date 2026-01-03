<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddNewKFTLTemplateStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide()"
            @requested_add_kftl_template_struct_element="(...kftl_template_struct_element: any[]) => emits('requested_add_kftl_template_struct_element', kftl_template_struct_element[0] as KFTLTemplateStructElementData)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewKFTLTemplateStructElementDialogEmits } from './add-new-kftl-template-struct-element-dialog-emits'
import type { AddNewKFTLTemplateStructElementDialogProps } from './add-new-kftl-template-struct-element-dialog-props'
import AddNewKFTLTemplateStructElementView from '../views/add-new-kftl_template-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data'

const add_new_kftl_template_struct_element_view = ref<InstanceType<typeof AddNewKFTLTemplateStructElementView> | null>(null);

defineProps<AddNewKFTLTemplateStructElementDialogProps>()
const emits = defineEmits<AddNewKFTLTemplateStructElementDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    add_new_kftl_template_struct_element_view.value?.reset_kftl_template_name()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    add_new_kftl_template_struct_element_view.value?.reset_kftl_template_name()
}
</script>
