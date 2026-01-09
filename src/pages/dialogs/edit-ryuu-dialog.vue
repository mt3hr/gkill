<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <v-card class="edit_ryuu_dialog_view">
            <RyuuListView v-model="model_value" :application_config="application_config" :gkill_api="gkill_api"
                :editable="true" :find_kyou_query_default="new FindKyouQuery()" :target_kyou="new Kyou()"
                @requested_apply_ryuu_struct="(...ryuu_data: any[]) => { emits('requested_apply_ryuu_struct', ryuu_data[0]) }"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_close_dialog="hide()" />
        </v-card>
    </v-dialog>
</template>
<script lang="ts" setup>
import { nextTick, type Ref, ref } from 'vue'
import Dnote from '../views/dnote-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import RyuuListView from '../views/ryuu-list-view.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { EditRyuuDialogEmits } from './edit-ryuu-dialog-emits'
import type { EditRyuuDialogProps } from './edit-ryuu-dialog-props'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { Kyou } from "@/classes/datas/kyou";
const dnote_view = ref<InstanceType<typeof Dnote> | null>(null);

const model_value = defineModel<ApplicationConfig>()
defineProps<EditRyuuDialogProps>()
const emits = defineEmits<EditRyuuDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    is_show_dialog.value = true
    nextTick(() => dnote_view.value?.reload([], new FindKyouQuery()))
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>

<style scoped lang="css">
.edit_ryuu_dialog_view {
    overflow-x: scroll;
}
</style>