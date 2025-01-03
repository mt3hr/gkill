<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>通知追加</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-textarea v-model="content_value" label="通知内容" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <label>通知日時</label>
                <input class="input date" type="date" v-model="notification_date" label="日付" />
                <input class="input time" type="time" v-model="notification_time" label="時刻" />
                <v-btn color="primary" @click="reset_notification_date_time()">リセット</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="save()">保存</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_plaing_end_button="true" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="false"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import type { AddNotificationViewProps } from './add-notification-view-props'
import { Notification } from '@/classes/datas/notification'
import { AddNotificationRequest } from '@/classes/api/req_res/add-notification-request'
import moment from 'moment'

const props = defineProps<AddNotificationViewProps>()
const emits = defineEmits<KyouViewEmits>()

const show_kyou: Ref<boolean> = ref(false)
const content_value: Ref<string> = ref("")
const notification_date: Ref<string> = ref("")
const notification_time: Ref<string> = ref("")

async function save(): Promise<void> {
    // 値がなかったらエラーメッセージを出力する
    if (content_value.value === "") {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "通知内容が未入力です"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    get_gkill_req.session_id = props.gkill_api.get_session_id()
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

    // 通知内容情報を用意する
    const new_notification = new Notification()
    new_notification.notification_time = moment(notification_date.value + " " + notification_time.value).toDate()
    new_notification.content = content_value.value
    new_notification.id = props.gkill_api.generate_uuid()
    new_notification.is_deleted = false
    new_notification.target_id = props.kyou.id
    new_notification.related_time = new Date(Date.now())
    new_notification.create_app = "gkill"
    new_notification.create_device = gkill_info_res.device
    new_notification.create_time = new Date(Date.now())
    new_notification.create_user = gkill_info_res.user_id
    new_notification.update_app = "gkill"
    new_notification.update_app = "gkill"
    new_notification.update_device = gkill_info_res.device
    new_notification.update_time = new Date(Date.now())
    new_notification.update_user = gkill_info_res.user_id
    new_notification.related_time = new Date(Date.now())

    // 追加リクエストを飛ばす
    const req = new AddNotificationRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.notification = new_notification
    const res = await props.gkill_api.add_notification(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_reload_kyou', props.kyou)
    emits('requested_close_dialog')
    return
}

function reset_notification_date_time(): void {
    notification_date.value = ""
    notification_time.value = ""
}
</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>