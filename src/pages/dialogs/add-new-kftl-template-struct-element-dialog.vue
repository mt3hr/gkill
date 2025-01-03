<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddNewKFTLTemplateStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide()"
            @requested_add_kftl_template_struct_element="(kftl_template_struct_element) => emits('requested_add_kftl_template_struct_element', kftl_template_struct_element)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewKFTLTemplateStructElementDialogEmits } from './add-new-kftl-template-struct-element-dialog-emits'
import type { AddNewKFTLTemplateStructElementDialogProps } from './add-new-kftl-template-struct-element-dialog-props'
import AddNewKFTLTemplateStructElementView from '../views/add-new-kftl_template-struct-element-view.vue'

const add_new_kftl_template_struct_element_view = ref<InstanceType<typeof AddNewKFTLTemplateStructElementView> | null>(null);

defineProps<AddNewKFTLTemplateStructElementDialogProps>()
const emits = defineEmits<AddNewKFTLTemplateStructElementDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    add_new_kftl_template_struct_element_view.value?.reset_kftl_template_name()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    add_new_kftl_template_struct_element_view.value?.reset_kftl_template_name()
}
</script>
