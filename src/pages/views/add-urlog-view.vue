<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ $t("ADD_URLOG_TITLE") }}</span>
                </v-col>
            </v-row>
        </v-card-title>
        <table>
            <tr>
                <td>
                    <label>{{ $t("URL_TITLE") }}</label>
                </td>
                <td>
                    <input class="input text" type="text" v-model="url" :label="$t('URL_TITLE')" autofocus
                        :readonly="is_requested_submit" />
                </td>
            </tr>
            <tr>
                <td>
                    <label>{{ $t("URLOG_TITLE_TITLE") }}</label>
                </td>
                <td>
                    <input class="input text" type="text" v-model="title" :label="$t('URLOG_TITLE_TITLE')"
                        :readonly="is_requested_submit" />
                </td>
            </tr>
        </table>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <label>{{ $t("URLOG_DATE_TIME_TITLE") }}</label>
                <input class="input date" type="date" v-model="related_date" :label="$t('URLOG_DATE_TITLE')"
                    :readonly="is_requested_submit" />
                <input class="input time" type="time" v-model="related_time" :label="$t('URLOG_TIME_TITLE')"
                    :readonly="is_requested_submit" />
                <v-btn dark color="secondary" @click="reset_related_date_time()" :disabled="is_requested_submit">{{
                    $t('RESET_TITLE') }}</v-btn>
                <v-btn dark color="primary" @click="now_to_related_date_time()" :disabled="is_requested_submit">{{
                    $t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()"
                    :disabled="is_requested_submit">{{ $t("RESET_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{ $t("SAVE_TITLE")
                    }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditURLogViewProps } from './edit-ur-log-view-props'
import { URLog } from '@/classes/datas/ur-log'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import type { KyouViewEmits } from './kyou-view-emits'
import { AddURLogRequest } from '@/classes/api/req_res/add-ur-log-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const is_requested_submit = ref(false)

const props = defineProps<EditURLogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const urlog: Ref<URLog> = ref((() => {
    const urlog = new URLog()
    urlog.related_time = new Date(Date.now())
    return urlog
})())
const title: Ref<string> = ref(urlog.value.title)
const url: Ref<string> = ref(urlog.value.url)
const related_date: Ref<string> = ref(moment(urlog.value.related_time).format("YYYY-MM-DD"))
const related_time: Ref<string> = ref(moment(urlog.value.related_time).format("HH:mm:ss"))

function reset(): void {
    title.value = urlog.value.title
    url.value = urlog.value.url
    related_date.value = moment(urlog.value.related_time).format("YYYY-MM-DD")
    related_time.value = moment(urlog.value.related_time).format("HH:mm:ss")
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        if (!urlog.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_urlog_is_null
            error.error_message = t("CLIENT_URLOG_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 日時必須入力チェック
        if (related_date.value === "" || related_time.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.urlog_related_time_is_blank
            error.error_message = t("URLOG_DATE_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // URL入力チェック
        if (url.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.urlog_url_is_blank
            error.error_message = t("URLOG_URL_IS_BLANK_MESSAGE")
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

        // 更新後URLog情報を用意する
        const new_urlog = await urlog.value.clone()
        new_urlog.id = props.gkill_api.generate_uuid()
        new_urlog.title = title.value
        new_urlog.url = url.value
        new_urlog.related_time = moment(related_date.value + " " + related_time.value).toDate()
        new_urlog.create_app = "gkill"
        new_urlog.create_device = gkill_info_res.device
        new_urlog.create_time = new Date(Date.now())
        new_urlog.create_user = gkill_info_res.user_id
        new_urlog.update_app = "gkill"
        new_urlog.update_device = gkill_info_res.device
        new_urlog.update_time = new Date(Date.now())
        new_urlog.update_user = gkill_info_res.user_id

        // 追加リクエストを飛ばす
        const req = new AddURLogRequest()
        req.urlog = new_urlog
        const res = await props.gkill_api.add_urlog(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits("updated_kyou", res.added_urlog_kyou)
        emits('requested_reload_list')
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}

function now_to_related_date_time(): void {
    related_date.value = moment().format("YYYY-MM-DD")
    related_time.value = moment().format("HH:mm:ss")
}

function reset_related_date_time(): void {
    related_date.value = moment(urlog.value.related_time).format("YYYY-MM-DD")
    related_time.value = moment(urlog.value.related_time).format("HH:mm:ss")
}
</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>