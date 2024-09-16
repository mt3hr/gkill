<template>
    <v-dialog v-model="is_show_dialog">
        <EditTagStructView :application_config="application_config" :gkill_api="gkill_api" :tag_struct="tag_struct"
            :tag_struct_root="tag_struct_root" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditTagStructDialogEmits } from './edit-tag-struct-dialog-emits';
import type { EditTagStructDialogProps } from './edit-tag-struct-dialog-props';
import EditTagStructView from '../views/edit-tag-struct-view.vue';

const props = defineProps<EditTagStructDialogProps>();
const emits = defineEmits<EditTagStructDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
