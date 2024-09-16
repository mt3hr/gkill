<template>
    <v-dialog v-model="is_show_dialog">
        <EditTagStructElementView :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :struct_obj="struct_obj" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_tag_struct_element="(tag_struct_element) => emits('requested_update_tag_struct_element', tag_struct_element)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditTagStructElementDialogEmits } from './edit-tag-struct-element-dialog-emits';
import type { EditTagStructElementDialogProps } from './edit-tag-struct-element-dialog-props';
import EditTagStructElementView from '../views/edit-tag-struct-element-view.vue';

const props = defineProps<EditTagStructElementDialogProps>();
const emits = defineEmits<EditTagStructElementDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
