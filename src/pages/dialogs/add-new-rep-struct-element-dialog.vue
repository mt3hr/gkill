<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddNewRepStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide"
            @requested_add_rep_struct_element="(rep_struct_element) => emits('requested_add_rep_struct_element', rep_struct_element)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { AddNewRepStructElementDialogEmits } from './add-new-rep-struct-element-dialog-emits'
import type { AddNewRepStructElementDialogProps } from './add-new-rep-struct-element-dialog-props'
import AddNewRepStructElementView from '../views/add-new-rep-struct-element-view.vue'

const add_new_rep_struct_element_view = ref<InstanceType<typeof AddNewRepStructElementView> | null>(null);

defineProps<AddNewRepStructElementDialogProps>()
const emits = defineEmits<AddNewRepStructElementDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    add_new_rep_struct_element_view.value?.reset_rep_name()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    add_new_rep_struct_element_view.value?.reset_rep_name()
}
</script>
