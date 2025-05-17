<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog" class="kyou_dialog">
        <v-card class="pa-2" :width="'fit-content'">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :is_readonly_mi_check="false"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :show_attached_timeis="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_related_time="true"
                @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
                @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
                @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
                @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
                @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
                @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
                @registered_text="(registered_text) => emits('registered_text', registered_text)"
                @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
                @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
                @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
                @updated_text="(updated_text) => emits('updated_text', updated_text)"
                @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
        </v-card>
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { KyouDialogProps } from './kyou-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import KyouView from '../views/kyou-view.vue'

defineProps<KyouDialogProps>()
const emits = defineEmits<KyouDialogEmits>()

defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
