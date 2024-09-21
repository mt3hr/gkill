<template>
    <v-dialog v-model="is_show_dialog">
        <AddNewRepTypeStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_rep_type_struct_element="(rep_type_struct) => emits('requested_add_rep_type_struct_element', rep_type_struct)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewRepTypeStructElementDialogEmits } from './add-new-rep-type-struct-element-dialog-emits'
import type { AddNewRepTypeStructElementDialogProps } from './add-new-rep-type-struct-element-dialog-props'
import AddNewRepTypeStructElementView from '../views/add-new-rep-type-struct-element-view.vue'

const props = defineProps<AddNewRepTypeStructElementDialogProps>()
const emits = defineEmits<AddNewRepTypeStructElementDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
