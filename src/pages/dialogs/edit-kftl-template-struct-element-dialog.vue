<template>
    <v-dialog v-model="is_show_dialog">
        <EditKFTLTemplateStructElementView :application_config="application_config" :folder_name="''"
            :gkill_api="gkill_api" :struct_obj="struct_obj"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_kftl_template_struct_element="(kftl_template_struct) => emits('requested_update_kftl_template_struct_element', kftl_template_struct)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditKFTLTemplateStructElementDialogEmits } from './edit-kftl-template-struct-element-dialog-emits';
import type { EditKFTLTemplateStructElementDialogProps } from './edit-kftl-template-struct-element-dialog-props';
import EditKFTLTemplateStructElementView from '../views/edit-kftl-template-struct-element-view.vue';

const props = defineProps<EditKFTLTemplateStructElementDialogProps>();
const emits = defineEmits<EditKFTLTemplateStructElementDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
