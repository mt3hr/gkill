<template>
    <v-dialog v-model="is_show_dialog">
        <EditTagStructElementView :application_config="application_config" :gkill_api="gkill_api"
            :struct_obj="tag_struct"
             @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_tag_struct="(tag_struct) => emits('requested_update_tag_struct', tag_struct)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditTagStructElementDialogEmits } from './edit-tag-struct-element-dialog-emits'
import type { EditTagStructElementDialogProps } from './edit-tag-struct-element-dialog-props'
import EditTagStructElementView from '../views/edit-tag-struct-element-view.vue'
import { TagStruct } from '@/classes/datas/config/tag-struct';

const props = defineProps<EditTagStructElementDialogProps>()
const emits = defineEmits<EditTagStructElementDialogEmits>()
defineExpose({ show, hide })

const tag_struct: Ref<TagStruct> = ref(new TagStruct())
const is_show_dialog: Ref<boolean> = ref(false)

async function show(tag_struct_obj: TagStruct): Promise<void> {
    tag_struct.value = tag_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
