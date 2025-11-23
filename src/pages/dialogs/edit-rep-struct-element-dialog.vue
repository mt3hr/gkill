<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditRepStructElementView :application_config="application_config" :gkill_api="gkill_api"
            :folder_name="i18n.global.t('REP_TITLE')" :struct_obj="rep_struct"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_rep_struct="(rep_struct) => emits('requested_update_rep_struct', rep_struct)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { EditRepStructElementDialogEmits } from './edit-rep-struct-element-dialog-emits'
import type { EditRepStructElementDialogProps } from './edit-rep-struct-element-dialog-props'
import EditRepStructElementView from '../views/edit-rep-struct-element-view.vue'
import { RepStruct } from '@/classes/datas/config/rep-struct';

defineProps<EditRepStructElementDialogProps>()
const emits = defineEmits<EditRepStructElementDialogEmits>()
defineExpose({ show, hide })

const rep_struct: Ref<RepStruct> = ref(new RepStruct())
const is_show_dialog: Ref<boolean> = ref(false)

async function show(rep_struct_obj: RepStruct): Promise<void> {
    rep_struct.value = rep_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
