<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddNewRepTypeStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide"
            @requested_add_rep_type_struct_element="(...rep_type_struct_element :any[]) => emits('requested_add_rep_type_struct_element', rep_type_struct_element[0] as RepTypeStructElementData)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { AddNewRepTypeStructElementDialogEmits } from './add-new-rep-type-struct-element-dialog-emits'
import type { AddNewRepTypeStructElementDialogProps } from './add-new-rep-type-struct-element-dialog-props'
import AddNewRepTypeStructElementView from '../views/add-new-rep-type-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'

const add_new_rep_type_struct_element_view = ref<InstanceType<typeof AddNewRepTypeStructElementView> | null>(null);

defineProps<AddNewRepTypeStructElementDialogProps>()
const emits = defineEmits<AddNewRepTypeStructElementDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    add_new_rep_type_struct_element_view.value?.reset_rep_type_name()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    add_new_rep_type_struct_element_view.value?.reset_rep_type_name()
}
</script>
