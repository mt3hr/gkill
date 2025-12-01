<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteRepStructView :application_config="application_config" :gkill_api="gkill_api"
            :rep_struct="rep_struct" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @requested_close_dialog="hide" @requested_delete_rep="(...id :any[]) => { if (id) emits('requested_delete_rep', id[0] as string); hide() }"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import ConfirmDeleteRepStructView from '../views/confirm-delete-rep-struct-view.vue';
import type { ConfirmDeleteRepStructDialogEmits } from './confirm-delete-rep-struct-dialog-emits.ts';
import type { ConfirmDeleteRepStructDialogProps } from './confirm-delete-rep-struct-dialog-props.ts';
import { RepStruct } from '@/classes/datas/config/rep-struct';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteRepStructDialogProps>()
const emits = defineEmits<ConfirmDeleteRepStructDialogEmits>()
defineExpose({ show, hide })

const rep_struct: Ref<RepStruct> = ref(new RepStruct())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(rep_struct_obj: RepStruct): Promise<void> {
    rep_struct.value = rep_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    rep_struct.value = new RepStruct()
}
</script>
