<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>通知削除</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        {{ notification.content }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="delete_notification()">削除</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="notification_highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="false" :show_attached_timeis="true"
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
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, nextTick, type Ref, ref } from 'vue'
import KyouView from './kyou-view.vue'
import type { InfoIdentifier } from '@/classes/datas/info-identifier';
import type { ConfirmDeleteNotificationViewProps } from './confirm-delete-notification-view-props';
import { UpdateNotificationRequest } from '@/classes/api/req_res/update-notification-request';

const props = defineProps<ConfirmDeleteNotificationViewProps>()
const emits = defineEmits<KyouViewEmits>()

const notification_highlight_targets = computed<Array<InfoIdentifier>>(() => {
    const info_identifer = props.notification.generate_info_identifer()
    return [info_identifer]
})

const show_kyou: Ref<boolean> = ref(true)

async function delete_notification(): Promise<void> {
    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

    // 更新後テキスト情報を用意する
    const updated_notification = await props.notification.clone()
    updated_notification.is_deleted = true
    updated_notification.update_app = "gkill"
    updated_notification.update_device = gkill_info_res.device
    updated_notification.update_time = new Date(Date.now())
    updated_notification.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateNotificationRequest()
    req.notification = updated_notification
    const res = await props.gkill_api.update_notification(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('deleted_notification', res.updated_notification)
    emits('requested_reload_kyou', props.kyou)
    emits('requested_close_dialog')
    return
}
</script>
