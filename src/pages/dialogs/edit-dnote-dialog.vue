<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <v-card>
            <Dnote :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api" :query="new FindKyouQuery()"
                :checked_kyous="[]" :last_added_tag="''" :editable="true"
                @received_messages="(messages) => emits('received_messages', messages)"
                @received_errors="(errors) => emits('received_errors', errors)" @requested_close_dialog="hide()"
                @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)"
                ref="dnote_view" />
        </v-card>
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import Dnote from '../views/dnote-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { type EditDnoteDialogProps } from './edit-dnote-dialog-props'
import { type EditDnoteDialogEmits } from './edit-dnote-dialog-emits'

defineProps<EditDnoteDialogProps>()
const emits = defineEmits<EditDnoteDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
