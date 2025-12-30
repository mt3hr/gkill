<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("EDIT_NOTIFICATION_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-textarea v-model="content_value" :label="i18n.global.t('NOTIFICATION_CONTENT_TITLE')" autofocus
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_notification_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="notification_date_string"
                                        :label="i18n.global.t('NOTIFICATION_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="notification_date_typed"
                                    @update:model-value="show_notification_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_notification_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="notification_time_string"
                                        :label="i18n.global.t('NOTIFICATION_TIME_TITLE')" readonly min-width="120"
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="notification_time_string" format="24hr"
                                    @update:minute="show_notification_time_menu = false" />
                            </v-menu>
                        </td>
                        <td>
                            <v-btn dark color="secondary" @click="reset_notification_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{
                    i18n.global.t("RESET_TITLE")
                    }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{
                    i18n.global.t("SAVE_TITLE")
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
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_update_time="false"
                :show_related_time="true" :show_attached_tags="true" :show_attached_texts="true"
                :show_attached_notifications="true"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { GkillError } from '@/classes/api/gkill-error';
import type { EditNotificationViewProps } from './edit-notification-view-props';
import { UpdateNotificationRequest } from '@/classes/api/req_res/update-notification-request';
import moment from 'moment';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache';
import type { GkillMessage } from '@/classes/api/gkill-message';
import type { Kyou } from '@/classes/datas/kyou';
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

const is_requested_submit = ref(false)

const props = defineProps<EditNotificationViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_notification: Ref<Notification> = ref(props.notification.clone())
const content_value: Ref<string> = ref(cloned_notification.value.content)
const notification_date_typed: Ref<Date> = ref(moment(props.notification.notification_time).toDate())
const notification_date_string: Ref<string> = computed(() => moment(notification_date_string.value).format("YYYY-MM-DD"))
const notification_time_string: Ref<string> = ref(moment(props.notification.notification_time).format("HH:mm:ss"))
const show_kyou: Ref<boolean> = ref(false)

const show_notification_date_menu = ref(false)
const show_notification_time_menu = ref(false)

watch(() => props.notification, () => load())
load()

async function load(): Promise<void> {
    cloned_notification.value = props.notification.clone()
    cloned_notification.value.attached_histories[0]
    content_value.value = cloned_notification.value.content
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // 日時必須入力チェック
        if (notification_date_string.value === "" || notification_time_string.value === "") {
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
            (moment(cloned_notification.value.notification_time).toDate().getTime() === moment(notification_date_string.value + " " + notification_time_string.value).toDate().getTime())) {
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
        updated_notification.notification_time = moment(notification_date_string.value + " " + notification_time_string.value).toDate()
        updated_notification.update_app = "gkill"
        updated_notification.update_device = gkill_info_res.device
        updated_notification.update_time = moment(new Date(Date.now())).toDate()
        updated_notification.update_user = gkill_info_res.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_notification.id)
        await delete_gkill_kyou_cache(updated_notification.target_id)
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
    notification_date_typed.value = moment(props.notification.notification_time).toDate()
    notification_time_string.value = moment(props.notification.notification_time).format("HH:mm:ss")
}

function reset_notification_date_time(): void {
    notification_date_typed.value = moment(props.notification.notification_time).toDate()
    notification_time_string.value = moment(props.notification.notification_time).format("HH:mm:ss")
}
</script>

<style lang="css" scoped></style>