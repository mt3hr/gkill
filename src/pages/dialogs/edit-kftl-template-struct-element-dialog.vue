<template>
    <v-dialog persistent @click:outside="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <EditKFTLTemplateStructElementView :application_config="application_config" :gkill_api="gkill_api"
            :struct_obj="kftl_template_struct"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_kftl_template_struct="(...kftl_template_struct: any[]) => emits('requested_update_kftl_template_struct', kftl_template_struct[0] as KFTLTemplateStructElementData)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditKFTLTemplateStructElementDialogEmits } from './edit-kftl-template-struct-element-dialog-emits'
import type { EditKFTLTemplateStructElementDialogProps } from './edit-kftl-template-struct-element-dialog-props'
import EditKFTLTemplateStructElementView from '../views/edit-kftl-template-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<EditKFTLTemplateStructElementDialogProps>()
const emits = defineEmits<EditKFTLTemplateStructElementDialogEmits>()
defineExpose({ show, hide })

const kftl_template_struct: Ref<KFTLTemplateStructElementData> = ref(new KFTLTemplateStructElementData())
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(kftl_template_struct_obj: KFTLTemplateStructElementData): Promise<void> {
    kftl_template_struct.value = kftl_template_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
