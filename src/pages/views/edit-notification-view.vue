<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ $t("EDIT_NOTIFICATION_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="$t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-textarea v-model="content_value" :label="$t('NOTIFICATION_CONTENT_TITLE')" autofocus
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <label>{{ $t("NOTIFICATION_DATE_TIME_TITLE") }}</label>
                <input class="input date" type="date" v-model="notification_date" :label="$t('NOTIFICATION_DATE_TITLE')"
                    :readonly="is_requested_submit" />
                <input class="input time" type="time" v-model="notification_time" :label="$t('NOTIFICATION_TIME_TITLE')"
                    :readonly="is_requested_submit" />
                <v-btn dark color="secondary" @click="reset_notification_date_time()" :disabled="is_requested_submit">{{
                    $t("RESET_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{ $t("RESET_TITLE")
                    }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{ $t("SAVE_TITLE")
                    }}</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="true" :show_attached_timeis="true"
                :show_related_time="true" @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
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
import { type Ref, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { GkillError } from '@/classes/api/gkill-error';
import type { EditNotificationViewProps } from './edit-notification-view-props';
import type { Notification } from '@/classes/datas/notification';
import { UpdateNotificationRequest } from '@/classes/api/req_res/update-notification-request';
import moment from 'moment';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';

import { i18n } from '@/i18n'

const is_requested_submit = ref(false)

const props = defineProps<EditNotificationViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_notification: Ref<Notification> = ref(props.notification.clone())
const content_value: Ref<string> = ref(cloned_notification.value.content)
const notification_date: Ref<string> = ref(moment(props.notification.notification_time).format("YYYY-MM-DD"))
const notification_time: Ref<string> = ref(moment(props.notification.notification_time).format("HH:mm:ss"))
const show_kyou: Ref<boolean> = ref(false)

watch(() => props.notification, () => load())
load()

async function load(): Promise<void> {
    cloned_notification.value = props.notification.clone()
    content_value.value = cloned_notification.value.content
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // 日時必須入力チェック
        if (notification_date.value === "" || notification_time.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.notification_time_is_blank
            error.error_message = i18n.global.t("NOTIFICATION_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 値がなかったらエラーメッセージを出力する
        if (content_value.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.notification_content_is_blank
            error.error_message = i18n.global.t("NOTIFICATION_CONTENT_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新がなかったらエラーメッセージを出力する
        if (cloned_notification.value.content === content_value.value &&
            (moment(cloned_notification.value.notification_time).toDate().getTime() === moment(notification_date.value + " " + notification_time.value).toDate().getTime())) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.notification_is_no_update
            error.error_message = i18n.global.t("NOTIFICATION_IS_NO_UPDATE_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // UserIDやDevice情報を取得する
        const get_gkill_req = new GetGkillInfoRequest()
        const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
        if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
            emits('received_errors', gkill_info_res.errors)
            return
        }

        // 更新後通知情報を用意する
        const updated_notification = await cloned_notification.value.clone()
        updated_notification.content = content_value.value
        updated_notification.notification_time = moment(notification_date.value + " " + notification_time.value).toDate()
        updated_notification.update_app = "gkill"
        updated_notification.update_device = gkill_info_res.device
        updated_notification.update_time = moment(new Date(Date.now())).toDate()
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
        emits("updated_notification", res.updated_notification)
        emits('requested_reload_kyou', props.kyou)
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}

function reset(): void {
    content_value.value = props.notification.content
    notification_date.value = moment(props.notification.notification_time).format("YYYY-MM-DD")
    notification_time.value = moment(props.notification.notification_time).format("HH:mm:ss")
}

function reset_notification_date_time(): void {
    notification_date.value = moment(props.notification.notification_time).format("YYYY-MM-DD")
    notification_time.value = moment(props.notification.notification_time).format("HH:mm:ss")
}
</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>