<template>
    <v-dialog v-model="is_show_dialog">
        <EditRepTypeStructElementView :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :struct_obj="struct_obj" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_rep_type_struct_element="(rep_type_struct) => emits('requested_update_rep_type_struct_element', rep_type_struct)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditRepTypeStructElementDialogEmits } from './edit-rep-type-struct-element-dialog-emits';
import type { EditRepTypeStructElementDialogProps } from './edit-rep-type-struct-element-dialog-props';
import EditRepTypeStructElementView from '../views/edit-rep-type-struct-element-view.vue';

const props = defineProps<EditRepTypeStructElementDialogProps>();
const emits = defineEmits<EditRepTypeStructElementDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
