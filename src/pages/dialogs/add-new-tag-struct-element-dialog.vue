<template>
    <v-dialog v-model="is_show_dialog">
        <AddNewTagStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide"
            @requested_add_tag_struct_element="(tag_struct_element) => emits('requested_add_tag_struct_element', tag_struct_element)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewTagStructElementDialogEmits } from './add-new-tag-struct-element-dialog-emits'
import type { AddNewTagStructElementDialogProps } from './add-new-tag-struct-element-dialog-props'
import AddNewTagStructElementView from '../views/add-new-tag-struct-element-view.vue'

const add_new_tag_struct_element_view = ref<InstanceType<typeof AddNewTagStructElementView> | null>(null);

const props = defineProps<AddNewTagStructElementDialogProps>()
const emits = defineEmits<AddNewTagStructElementDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    add_new_tag_struct_element_view.value?.reset_tag_name()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    add_new_tag_struct_element_view.value?.reset_tag_name()
}
</script>
