<template>
    <v-card class="pa-0 ma-0">
        <v-textarea v-model="content_value" :label="$t('NOTIFICATION_CONTENT_TITLE')" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <label>{{ $t("NOTIFICATION_DATE_TIME_TITLE") }}</label>
                <input class="input date" type="date" v-model="notification_date"
                    :label="$t('NOTIFICATION_DATE_TITLE')" />
                <input class="input time" type="time" v-model="notification_time"
                    :label="$t('NOTIFICATION_TIME_TITLE')" />
                <v-btn dark color="secondary" @click="reset_notification_date_time()">{{ $t("RESET_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { Notification } from '@/classes/datas/notification'
import moment from 'moment'
import type { AddNotificationForAddMiViewProps } from './add-notification-for-add-mi-view-props'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<AddNotificationForAddMiViewProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ get_notification })

const content_value: Ref<string> = ref(props.default_notification ? props.default_notification.content : "")
const notification_date: Ref<string> = ref(props.default_notification && props.default_notification.notification_time.getTime() !== new Date(0).getTime() ? moment(props.default_notification.notification_time).format("YYYY-MM-DD") : "")
const notification_time: Ref<string> = ref(props.default_notification && props.default_notification.notification_time.getTime() !== new Date(0).getTime() ? moment(props.default_notification.notification_time).format("HH:mm:ss") : "")

async function get_notification(): Promise<Notification | null> {
    // 値がなかったらエラーメッセージを出力する
    if (content_value.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.notification_content_is_blank
        error.error_message = t("NOTIFICATION_CONTENT_IS_BLANK_MESSAGE")
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return null
    }
    // 通知日時 入力なしエラーチェック
    if (notification_date.value === "" || notification_time.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.notification_time_is_blank
        error.error_message = t("NOTIFICATION_DATE_TIME_IS_BLANK_MESSAGE")
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return null
    }

    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return null
    }

    // 通知情報を用意する
    const new_notification = new Notification()
    new_notification.notification_time = moment(notification_date.value + " " + notification_time.value).toDate()
    new_notification.content = content_value.value
    new_notification.id = props.default_notification ? props.default_notification.id : props.gkill_api.generate_uuid()
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
    return new_notification
}

function reset_notification_date_time(): void {
    notification_date.value = props.default_notification && props.default_notification.notification_time.getTime() !== new Date(0).getTime() ? moment(props.default_notification.notification_time).format("YYYY-MM-DD") : ""
    notification_time.value = props.default_notification && props.default_notification.notification_time.getTime() !== new Date(0).getTime() ? moment(props.default_notification.notification_time).format("HH:mm:ss") : ""
}
</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>