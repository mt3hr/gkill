<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddNewFoloderView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide"
            @requested_add_new_folder="(new_folder) => emits('requested_add_new_folder', new_folder)"
            ref="add_new_folder_view" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewFoloderDialogEmits } from './add-new-foloder-dialog-emits'
import type { AddNewFoloderDialogProps } from './add-new-foloder-dialog-props'
import AddNewFoloderView from '../views/add-new-foloder-view.vue'

const add_new_folder_view = ref<InstanceType<typeof AddNewFoloderView> | null>(null);

defineProps<AddNewFoloderDialogProps>()
const emits = defineEmits<AddNewFoloderDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    add_new_folder_view.value?.reset_folder_name()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    add_new_folder_view.value?.reset_folder_name()
}
</script>
