<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <EditTagStructElementView :application_config="application_config" :gkill_api="gkill_api"
            :struct_obj="tag_struct"
             @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_tag_struct="(...tag_struct :any[]) => emits('requested_update_tag_struct', tag_struct[0] as TagStructElementData)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditTagStructElementDialogEmits } from './edit-tag-struct-element-dialog-emits'
import type { EditTagStructElementDialogProps } from './edit-tag-struct-element-dialog-props'
import EditTagStructElementView from '../views/edit-tag-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<EditTagStructElementDialogProps>()
const emits = defineEmits<EditTagStructElementDialogEmits>()
defineExpose({ show, hide })

const tag_struct: Ref<TagStructElementData> = ref(new TagStructElementData())
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(tag_struct_obj: TagStructElementData): Promise<void> {
    tag_struct.value = tag_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
