<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <v-card class="edit_ryuu_dialog_view">
            <RyuuListView v-model="model_value" :application_config="application_config" :gkill_api="gkill_api"
                :editable="true" :find_kyou_query_default="new FindKyouQuery()" :related_time="new Date(Date.now())"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_close_dialog="hide()" />
        </v-card>
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref } from 'vue'
import Dnote from '../views/dnote-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import RyuuListView from '../views/ryuu-list-view.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { EditRyuuDialogEmits } from './edit-ryuu-dialog-emits'
import type { EditRyuuDialogProps } from './edit-ryuu-dialog-props'
const dnote_view = ref<InstanceType<typeof Dnote> | null>(null);

const model_value = defineModel<ApplicationConfig>()
defineProps<EditRyuuDialogProps>()
const emits = defineEmits<EditRyuuDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

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