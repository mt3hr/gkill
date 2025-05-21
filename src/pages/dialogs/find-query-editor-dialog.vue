<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <v-card>
            <FindQueryEditorView :application_config="received_application_config" :gkill_api="gkill_api"
                :find_kyou_query="model_value!" :inited="inited"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide()"
                @requested_apply="(find_kyou_query) => model_value = find_kyou_query" ref="find_query_editor_view" />
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import { nextTick, ref, type Ref } from 'vue'
import FindQueryEditorView from '../views/find-query-editor-view.vue';
import type FindQueryEditorDialogProps from './find-query-editor-dialog-props';
import type FindQueryEditorDialogEmits from './find-query-editor-dialog-emits';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import { ApplicationConfig } from '@/classes/datas/config/application-config';
const is_show_dialog: Ref<boolean> = ref(false)
const inited = ref(false)

const model_value = defineModel<FindKyouQuery>()
defineExpose({ show, hide })
const props = defineProps<FindQueryEditorDialogProps>()
const emits = defineEmits<FindQueryEditorDialogEmits>()

const received_application_config = ref(new ApplicationConfig())

async function show(): Promise<void> {
    is_show_dialog.value = true
    received_application_config.value = new ApplicationConfig()
    nextTick(() => received_application_config.value = props.application_config) // TODO なんかApplicationConfigが切り替わったタイミングでQueryEditorが読み込まれるっぽい・・・
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
