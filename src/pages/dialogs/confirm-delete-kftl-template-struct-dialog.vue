<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteKFTLTemplateStructView :application_config="application_config" :gkill_api="gkill_api"
            :kftl_template_struct="kftl_template_struct" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @requested_close_dialog="hide"
            @requested_delete_kftl_template="(...id :any[]) => { emits('requested_delete_kftl_template', id[0] as string); hide() }"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import ConfirmDeleteKFTLTemplateStructView from '../views/confirm-delete-kftl-template-struct-view.vue';
import type { ConfirmDeleteKFTLTemplateStructDialogEmits } from './confirm-delete-kftl-template-struct-dialog-emits.ts';
import type { ConfirmDeleteKFTLTemplateStructDialogProps } from './confirm-delete-kftl-template-struct-dialog-props.ts';
import { KFTLTemplateStruct } from '@/classes/datas/config/kftl-template-struct';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteKFTLTemplateStructDialogProps>()
const emits = defineEmits<ConfirmDeleteKFTLTemplateStructDialogEmits>()
defineExpose({ show, hide })

const kftl_template_struct: Ref<KFTLTemplateStruct> = ref(new KFTLTemplateStruct())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(kftl_template_struct_obj: KFTLTemplateStruct): Promise<void> {
    kftl_template_struct.value = kftl_template_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    kftl_template_struct.value = new KFTLTemplateStruct()
}
</script>
