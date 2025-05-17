<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteTagStructView :application_config="application_config" :gkill_api="gkill_api"
            :tag_struct="tag_struct" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_close_dialog="hide" @requested_delete_tag="(id) => { emits('requested_delete_tag', id); hide() }"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import ConfirmDeleteTagStructView from '../views/confirm-delete-tag-struct-view.vue';
import type { ConfirmDeleteTagStructDialogEmits } from './confirm-delete-tag-struct-dialog-emits';
import type { ConfirmDeleteTagStructDialogProps } from './confirm-delete-tag-struct-dialog-props';
import { TagStruct } from '@/classes/datas/config/tag-struct';

defineProps<ConfirmDeleteTagStructDialogProps>()
const emits = defineEmits<ConfirmDeleteTagStructDialogEmits>()
defineExpose({ show, hide })

const tag_struct: Ref<TagStruct> = ref(new TagStruct())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(tag_struct_obj: TagStruct): Promise<void> {
    tag_struct.value = tag_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    tag_struct.value = new TagStruct()
}
</script>
