<template>
    <v-dialog v-model="is_show_dialog">
        <AddNewRepStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_rep_struct_element="(rep_struct) => emits('requested_add_rep_struct_element', rep_struct)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { AddNewRepStructElementDialogEmits } from './add-new-rep-struct-element-dialog-emits';
import type { AddNewRepStructElementDialogProps } from './add-new-rep-struct-element-dialog-props';
import AddNewRepStructElementView from '../views/add-new-rep-struct-element-view.vue';

const props = defineProps<AddNewRepStructElementDialogProps>();
const emits = defineEmits<AddNewRepStructElementDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
