<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddNewTagStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide"
            @requested_add_tag_struct_element="(...tag_struct_element :any[]) => emits('requested_add_tag_struct_element', tag_struct_element[0] as TagStructElementData)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { AddNewTagStructElementDialogEmits } from './add-new-tag-struct-element-dialog-emits'
import type { AddNewTagStructElementDialogProps } from './add-new-tag-struct-element-dialog-props'
import AddNewTagStructElementView from '../views/add-new-tag-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'

const add_new_tag_struct_element_view = ref<InstanceType<typeof AddNewTagStructElementView> | null>(null);

defineProps<AddNewTagStructElementDialogProps>()
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
