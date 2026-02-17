<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteTagStructView :application_config="application_config" :gkill_api="gkill_api"
            :tag_struct="tag_struct"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @requested_close_dialog="hide"
            @requested_delete_tag="(...id: any[]) => { emits('requested_delete_tag', id[0] as string); hide() }"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import ConfirmDeleteTagStructView from '../views/confirm-delete-tag-struct-view.vue';
import type { ConfirmDeleteTagStructDialogEmits } from './confirm-delete-tag-struct-dialog-emits';
import type { ConfirmDeleteTagStructDialogProps } from './confirm-delete-tag-struct-dialog-props';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteTagStructDialogProps>()
const emits = defineEmits<ConfirmDeleteTagStructDialogEmits>()
defineExpose({ show, hide })

const tag_struct: Ref<TagStructElementData> = ref(new TagStructElementData())

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data';
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(tag_struct_obj: TagStructElementData): Promise<void> {
    tag_struct.value = tag_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    tag_struct.value = new TagStructElementData()
}
</script>
