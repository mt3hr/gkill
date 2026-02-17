<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <v-card>
            <FindQueryEditorView v-if="model_value" :application_config="received_application_config"
                :gkill_api="gkill_api" :find_kyou_query="model_value" :inited="inited"
                @updated_query="(...query: any[]) => model_value = (query[0] as FindKyouQuery)" @inited="inited = true"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_close_dialog="hide()" @requested_apply="(...find_kyou_query :any[])  => model_value = find_kyou_query[0] as FindKyouQuery"
                ref="find_query_editor_view" />
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { nextTick, ref, watch, type Ref } from 'vue'
import FindQueryEditorView from '../views/find-query-editor-view.vue';
import type FindQueryEditorDialogProps from './find-query-editor-dialog-props';
import type FindQueryEditorDialogEmits from './find-query-editor-dialog-emits';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import { ApplicationConfig } from '@/classes/datas/config/application-config';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
const inited = ref(false)

const model_value = defineModel<FindKyouQuery>()
defineExpose({ show, hide })
const props = defineProps<FindQueryEditorDialogProps>()
const emits = defineEmits<FindQueryEditorDialogEmits>()
const cloned_find_kyou_query = ref<FindKyouQuery | null>(null)

watch(() => inited.value, () => {
    if (inited.value) {
        return nextTick(async () => {
            model_value.value = cloned_find_kyou_query.value!
        })
    }
})

const received_application_config = ref(new ApplicationConfig())

async function show(find_kyou_query: FindKyouQuery): Promise<void> {
    return nextTick(async () => {
        cloned_find_kyou_query.value = find_kyou_query
        cloned_find_kyou_query.value.query_id = props.gkill_api.generate_uuid()
        is_show_dialog.value = true
        received_application_config.value = new ApplicationConfig()
        await nextTick(() => received_application_config.value = props.application_config) // TODO なんかApplicationConfigが切り替わったタイミングでQueryEditorが読み込まれるっぽい・・・
    })
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
