<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteRepTypeStructView :application_config="application_config" :gkill_api="gkill_api"
            :rep_type_struct="rep_type_struct" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @requested_close_dialog="hide"
            @requested_delete_rep_type="(...id :any[]) => { emits('requested_delete_rep_type', id[0] as string); hide() }"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import ConfirmDeleteRepTypeStructView from '../views/confirm-delete-rep-type-struct-view.vue';
import type { ConfirmDeleteRepTypeStructDialogEmits } from './confirm-delete-rep-type-struct-dialog-emits';
import type { ConfirmDeleteRepTypeStructDialogProps } from './confirm-delete-rep-type-struct-dialog-props';
import { RepTypeStruct } from '@/classes/datas/config/rep-type-struct';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteRepTypeStructDialogProps>()
const emits = defineEmits<ConfirmDeleteRepTypeStructDialogEmits>()
defineExpose({ show, hide })

const rep_type_struct: Ref<RepTypeStruct> = ref(new RepTypeStruct())

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(rep_type_struct_obj: RepTypeStruct): Promise<void> {
    rep_type_struct.value = rep_type_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    rep_type_struct.value = new RepTypeStruct()
}
</script>
