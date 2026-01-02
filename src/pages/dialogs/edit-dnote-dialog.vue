<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <v-card class="edit_dnote_dialog_view">
            <Dnote :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api" :query="new FindKyouQuery()"
                :checked_kyous="[]" :last_added_tag="''" :editable="true"
                @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)" @requested_close_dialog="hide()"
                @requested_reload_application_config="(...application_config :any[]) => emits('requested_reload_application_config', application_config [0] as ApplicationConfig)"
                ref="dnote_view" />
        </v-card>
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref } from 'vue'
import Dnote from '../views/dnote-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { type EditDnoteDialogProps } from './edit-dnote-dialog-props'
import { type EditDnoteDialogEmits } from './edit-dnote-dialog-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
const dnote_view = ref<InstanceType<typeof Dnote> | null>(null);

defineProps<EditDnoteDialogProps>()
const emits = defineEmits<EditDnoteDialogEmits>()
defineExpose({ show, hide })

import { useBackToCloseDialog } from '@/classes/use-back-to-close-dialog'
const is_show_dialog: Ref<boolean> = ref(false)
useBackToCloseDialog(is_show_dialog)

async function show(): Promise<void> {
    is_show_dialog.value = true
    nextTick(() => dnote_view.value?.reload([], new FindKyouQuery()))
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>

<style lang="css" scoped>
.edit_dnote_dialog_view {
    overflow-x: scroll;
}
</style>