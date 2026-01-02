<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddNewFoloderView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide"
            @requested_add_new_folder="(...new_folder :any[]) => emits('requested_add_new_folder', new_folder[0] as FolderStructElementData)"
            ref="add_new_folder_view" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { AddNewFoloderDialogEmits } from './add-new-foloder-dialog-emits'
import type { AddNewFoloderDialogProps } from './add-new-foloder-dialog-props'
import AddNewFoloderView from '../views/add-new-foloder-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'

const add_new_folder_view = ref<InstanceType<typeof AddNewFoloderView> | null>(null);

defineProps<AddNewFoloderDialogProps>()
const emits = defineEmits<AddNewFoloderDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    add_new_folder_view.value?.reset_folder_name()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    add_new_folder_view.value?.reset_folder_name()
}
</script>
