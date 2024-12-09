<template>
    <v-dialog v-model="is_show_dialog">
        <v-card class="pa-2">
            <v-card-title>
                <v-row class="pa-0 ma-0">
                    <v-col cols="auto" class="pa-0 ma-0">
                        <span>タグ履歴</span>
                    </v-col>
                    <v-spacer />
                    <v-col cols="auto" class="pa-0 ma-0">
                        <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                    </v-col>
                </v-row>
            </v-card-title>
            <tagHistoriesView :application_config="application_config" :gkill_api="gkill_api" :tag="tag" :kyou="kyou"
                :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
                :highlight_targets="highlight_targets"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
            <v-card v-if="show_kyou">
                <KyouView :application_config="application_config" :gkill_api="gkill_api"
                    :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                    :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                    :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                    :show_mi_limit_time="true" :show_timeis_plaing_end_button="true" :height="'100%'" :width="'100%'"
                    @received_errors="(errors) => emits('received_errors', errors)"
                    @received_messages="(messages) => emits('received_messages', messages)"
                    @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                    @requested_reload_list="() => { }"
                    @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
            </v-card>
        </v-card>
    </v-dialog>
</template>
<script setup lang="ts">
import type { TagHistoriesDialogProps } from './tag-histories-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import { type Ref, ref } from 'vue'
import KyouView from '../views/kyou-view.vue'
import tagHistoriesView from '../views/tag-histories-view.vue'

const props = defineProps<TagHistoriesDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
const show_kyou: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
