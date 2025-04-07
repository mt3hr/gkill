<template>
    <v-card v-if="cloned_kyou.typed_timeis" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ $t("END_TIMEIS_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="$t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>{{ $t("END_TIMEIS_TITLE_TITLE") }}</label>
            </v-col>
            <v-col cols="auto">
                <input readonly type="text" v-model="timeis_title" :label="$t('END_TIMEIS_TITLE_TITLE')" autofocus />
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>{{ $t("START_DATE_TIME_TITLE") }}</label>
            </v-col>
            <v-col cols="auto">
                <input readonly type="date" v-model="timeis_start_date" :label="$t('START_DATE_TITLE')" />
                <input readonly type="time" v-model="timeis_start_time" :label="$t('START_TIME_TITLE')" />
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>{{ $t("END_DATE_TIME_TITLE") }}</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="timeis_end_date" :label="$t('END_DATE_TITLE')"
                    :readonly="is_requested_submit" />
                <input class="input date" type="time" v-model="timeis_end_time" :label="$t('END_TIME_TITLE')"
                    :readonly="is_requested_submit" />
            </v-col>
            <v-col cols="auto">
                <v-btn dark color="secondary" @click="clear_end_date_time()" :disabled="is_requested_submit">{{
                    $t("CLEAR_TITLE") }}</v-btn>
                <v-btn dark color="primary" @click="now_to_end_date_time()" :disabled="is_requested_submit">{{
                    $t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{ $t("RESET_TITLE")
                    }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{ $t("RESET_TITLE")
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
import type { EndTimeIsPlaingViewProps } from './end-time-is-plaing-view-props'
import { nextTick, type Ref, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import KyouView from './kyou-view.vue'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateTimeisRequest } from '@/classes/api/req_res/update-timeis-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const is_requested_submit = ref(false)

const props = defineProps<EndTimeIsPlaingViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const timeis_title: Ref<string> = ref(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : "")
const timeis_start_date: Ref<string> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.start_time).format("YYYY-MM-DD") : "")
const timeis_start_time: Ref<string> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.start_time).format("HH:mm:ss") : "")
const timeis_end_date: Ref<string> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.end_time).format("YYYY-MM-DD") : "")
const timeis_end_time: Ref<string> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.end_time).format("HH:mm:ss") : "")

const show_kyou: Ref<boolean> = ref(false)

watch(() => props.kyou, () => load())
load()

async function load(): Promise<void> {
    cloned_kyou.value = props.kyou.clone()
    await cloned_kyou.value.load_typed_datas()
    cloned_kyou.value.load_all()
    timeis_title.value = cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : ""
    timeis_start_date.value = moment(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.start_time : "").format("YYYY-MM-DD")
    timeis_start_time.value = moment(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.start_time : "").format("HH:mm:ss")
    timeis_end_date.value = moment().format("YYYY-MM-DD")
    timeis_end_time.value = moment().format("HH:mm:ss")
}

function reset(): void {
    timeis_title.value = cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : ""
    reset_start_date_time()
    reset_end_date_time()
}

function reset_start_date_time(): void {
    timeis_start_date.value = cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis.start_time).format("YYYY-MM-DD") : ""
    timeis_start_time.value = cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis.start_time).format("HH:mm:ss") : ""
}

function reset_end_date_time(): void {
    timeis_end_date.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).format("YYYY-MM-DD") : ""
    timeis_end_time.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).format("HH:mm:ss") : ""
}

function clear_end_date_time(): void {
    timeis_end_date.value = ""
    timeis_end_time.value = ""
}

function now_to_end_date_time(): void {
    timeis_end_date.value = moment().format("YYYY-MM-DD")
    timeis_end_time.value = moment().format("HH:mm:ss")
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        const timeis = cloned_kyou.value.typed_timeis
        if (!timeis) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_timeis_is_null
            error.error_message = t("CLIENT_TIMEIS_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // タイトル入力チェック
        if (timeis_title.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.timeis_title_is_blank
            error.error_message = t("TIMEIS_TITLE_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 開始日時必須入力チェック
        if (timeis_start_date.value === "" || timeis_start_time.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.timeis_start_time_is_blank
            error.error_message = t("TIMEIS_START_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 終了日時入力チェック
        if ((timeis_end_date.value === "" && timeis_end_time.value !== "") ||
            (timeis_end_date.value !== "" && timeis_end_time.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
            const error = new GkillError()
            error.error_code = GkillErrorCodes.timeis_end_time_is_blank
            error.error_message = t("TIMEIS_END_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新がなかったらエラーメッセージを出力する
        if (timeis.title === timeis_title.value &&
            (moment(timeis.start_time).toDate().getTime() === moment(timeis_start_date.value + " " + timeis_start_time.value).toDate().getTime()) &&
            (moment(timeis.end_time).toDate().getTime() === moment(timeis_end_date.value + " " + timeis_end_time.value).toDate().getTime()) || (timeis.end_time === null && timeis_end_date.value === "" && timeis_end_time.value === "")) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.timeis_is_no_update
            error.error_message = t("TIMEIS_IS_NO_UPDATE_MESSAGE")
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

        // 更新後TimeIs情報を用意する
        let end_time: Date | null = null
        if (timeis_end_date.value !== "" && timeis_end_time.value !== "") {
            end_time = moment(timeis_end_date.value + " " + timeis_end_time.value).toDate()
        }
        const updated_timeis = await timeis.clone()
        updated_timeis.title = timeis_title.value
        updated_timeis.start_time = moment(timeis_start_date.value + " " + timeis_start_time.value).toDate()
        updated_timeis.end_time = end_time
        updated_timeis.update_app = "gkill"
        updated_timeis.update_device = gkill_info_res.device
        updated_timeis.update_time = new Date(Date.now())
        updated_timeis.update_user = gkill_info_res.user_id

        // 更新リクエストを飛ばす
        const req = new UpdateTimeisRequest()
        req.timeis = updated_timeis

        const res = await props.gkill_api.update_timeis(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits("updated_kyou", res.updated_timeis_kyou)
        emits('requested_reload_kyou', props.kyou)
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}
</script>
<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>