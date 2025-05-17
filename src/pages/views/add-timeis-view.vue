<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_TIMEIS_TITLE") }}</span>
                </v-col>
                <v-spacer />
            </v-row>
        </v-card-title>
        <v-text-field class="input text" type="text" v-model="timeis_title" :label="i18n.global.t('TIMEIS_TITLE_TITLE')"
            autofocus :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_start_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="timeis_start_date_string"
                                        :label="i18n.global.t('TIMEIS_START_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="timeis_start_date_typed"
                                    @update:model-value="show_start_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_start_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="timeis_start_time_string"
                                        :label="i18n.global.t('TIMEIS_START_TIME_TITLE')" min-width="120" readonly
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="timeis_start_time_string" format="24hr"
                                    @update:model-value="show_start_time_menu = false" />
                            </v-menu>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="reset_start_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_start_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_end_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="timeis_end_date_string"
                                        :label="i18n.global.t('TIMEIS_END_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="timeis_end_date_typed"
                                    @update:model-value="show_end_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_end_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="timeis_end_time_string"
                                        :label="i18n.global.t('TIMEIS_END_TIME_TITLE')" min-width="120" readonly
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="timeis_end_time_string" format="24hr"
                                    @update:model-value="show_end_time_menu = false" />
                            </v-menu>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="reset_end_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_end_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
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
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditTimeIsViewProps } from './edit-time-is-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { TimeIs } from '@/classes/datas/time-is'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { AddTimeisRequest } from '@/classes/api/req_res/add-timeis-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/labs/components'

const is_requested_submit = ref(false)

const props = defineProps<EditTimeIsViewProps>()
const emits = defineEmits<KyouViewEmits>()

const timeis: Ref<TimeIs> = ref(((): TimeIs => {
    const timeis = new TimeIs()
    timeis.start_time = moment().toDate()
    timeis.end_time = null
    return timeis
})())
const timeis_title: Ref<string> = ref(timeis.value.title)
const timeis_start_date_typed: Ref<Date> = ref(moment(timeis.value.start_time).toDate())
const timeis_start_date_string: Ref<string> = computed(() => moment(timeis_start_date_typed.value).format("YYYY-MM-DD"))
const timeis_start_time_string: Ref<string> = ref(moment(timeis.value.start_time).format("HH:mm:ss"))
const timeis_end_date_typed: Ref<Date | null> = ref(null)
const timeis_end_date_string: Ref<string> = computed(() => timeis_end_date_typed.value ? moment(timeis_end_date_typed.value).format("YYYY-MM-DD") : "")
const timeis_end_time_string: Ref<string> = ref("")

const show_start_date_menu = ref(false)
const show_start_time_menu = ref(false)
const show_end_date_menu = ref(false)
const show_end_time_menu = ref(false)

function reset(): void {
    timeis_title.value = (timeis.value.title)
    timeis_start_date_string.value = (moment(timeis.value.start_time).format("YYYY-MM-DD"))
    timeis_start_time_string.value = (moment(timeis.value.start_time).format("HH:mm:ss"))
    timeis_end_date_typed.value = null
    timeis_end_time_string.value = ""
}

function reset_start_date_time(): void {
    timeis_start_date_typed.value = moment(timeis.value.start_time).toDate()
    timeis_start_time_string.value = moment(timeis.value.start_time).format("HH:mm:ss")
}

function reset_end_date_time(): void {
    timeis_end_date_typed.value = null
    timeis_end_time_string.value = ""
}

function clear_end_date_time(): void {
    timeis_end_date_typed.value = null
    timeis_end_time_string.value = ""
}

function now_to_start_date_time(): void {
    timeis_start_date_typed.value = moment().toDate()
    timeis_start_time_string.value = moment().format("HH:mm:ss")
}

function now_to_end_date_time(): void {
    timeis_end_date_typed.value = moment().toDate()
    timeis_end_time_string.value = moment().format("HH:mm:ss")
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        if (!timeis.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_timeis_is_null
            error.error_message = i18n.global.t("CLIENT_TIMEIS_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 開始日時必須入力チェック
        if (timeis_start_date_string.value === "" || timeis_start_time_string.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.timeis_start_time_is_blank
            error.error_message = i18n.global.t("TIMEIS_START_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 終了日時 片方だけ入力されていたらエラーチェック
        if (timeis_end_date_string.value === "" || timeis_end_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
            if ((timeis_end_date_string.value === "" && timeis_end_time_string.value !== "") ||
                (timeis_end_date_string.value !== "" && timeis_end_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                const error = new GkillError()
                error.error_code = GkillErrorCodes.timeis_end_time_is_blank
                error.error_message = i18n.global.t("TIMEIS_END_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }
        }

        // タイトル入力チェック
        if (timeis_title.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.timeis_title_is_blank
            error.error_message = i18n.global.t("TIMEIS_TITLE_IS_BLANK_MESSAGE")
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
        if (timeis_end_date_string.value !== "" && timeis_end_time_string.value !== "") {
            end_time = moment(timeis_end_date_string.value + " " + timeis_end_time_string.value).toDate()
        }
        const new_timeis = await timeis.value.clone()
        new_timeis.id = props.gkill_api.generate_uuid()
        new_timeis.title = timeis_title.value
        new_timeis.start_time = moment(timeis_start_date_string.value + " " + timeis_start_time_string.value).toDate()
        new_timeis.end_time = end_time
        new_timeis.create_app = "gkill"
        new_timeis.create_device = gkill_info_res.device
        new_timeis.create_time = new Date(Date.now())
        new_timeis.create_user = gkill_info_res.user_id
        new_timeis.update_app = "gkill"
        new_timeis.update_device = gkill_info_res.device
        new_timeis.update_time = new Date(Date.now())
        new_timeis.update_user = gkill_info_res.user_id

        // 追加リクエストを飛ばす
        const req = new AddTimeisRequest()
        req.timeis = new_timeis
        const res = await props.gkill_api.add_timeis(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits("updated_kyou", res.added_timeis_kyou)
        emits('requested_reload_list')
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}


</script>

<style lang="css" scoped></style>