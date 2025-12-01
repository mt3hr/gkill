<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditTagStructView :application_config="application_config" :gkill_api="gkill_api"
            :tag_struct="application_config.tag_struct" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_application_config="emits('requested_reload_application_config')"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { EditTagStructDialogEmits } from './edit-tag-struct-dialog-emits'
import type { EditTagStructDialogProps } from './edit-tag-struct-dialog-props'
import EditTagStructView from '../views/edit-tag-struct-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<EditTagStructDialogProps>()
const emits = defineEmits<EditTagStructDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}

</script>
