<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditRepTypeStructElementView :application_config="application_config" :gkill_api="gkill_api"
            :struct_obj="rep_type_struct" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_rep_type_struct="(...rep_type_struct :any[]) => emits('requested_update_rep_type_struct', rep_type_struct[0] as RepTypeStruct)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { EditRepTypeStructElementDialogEmits } from './edit-rep-type-struct-element-dialog-emits'
import type { EditRepTypeStructElementDialogProps } from './edit-rep-type-struct-element-dialog-props'
import EditRepTypeStructElementView from '../views/edit-rep-type-struct-element-view.vue'
import { RepTypeStruct } from '@/classes/datas/config/rep-type-struct';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<EditRepTypeStructElementDialogProps>()
const emits = defineEmits<EditRepTypeStructElementDialogEmits>()
defineExpose({ show, hide })

const rep_type_struct: Ref<RepTypeStruct> = ref(new RepTypeStruct())
const is_show_dialog: Ref<boolean> = ref(false)

async function show(rep_type_struct_obj: RepTypeStruct): Promise<void> {
    rep_type_struct.value = rep_type_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
