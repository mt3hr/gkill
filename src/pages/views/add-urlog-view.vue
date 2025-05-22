<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_URLOG_TITLE") }}</span>
                </v-col>
            </v-row>
        </v-card-title>
        <table>
            <tr>
                <td>
                    <v-text-field class="input text" type="text" v-model="url" :label="i18n.global.t('URL_TITLE')"
                        autofocus :readonly="is_requested_submit" />
                </td>
            </tr>
            <tr>
                <td>
                    <v-text-field class="input text" type="text" v-model="title"
                        :label="i18n.global.t('URLOG_TITLE_TITLE')" :readonly="is_requested_submit" />
                </td>
            </tr>
        </table>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_related_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_date_string"
                                        :label="i18n.global.t('URLOG_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="related_date_typed"
                                    @update:model-value="show_related_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_related_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_time_string"
                                        :label="i18n.global.t('URLOG_TIME_TITLE')" readonly min-width="120"
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="related_time_string" format="24hr"
                                    @update:model-value="show_related_time_menu = false" />
                            </v-menu>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="pt-2">
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="reset_related_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_related_date_time()"
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
import type { EditURLogViewProps } from './edit-ur-log-view-props'
import { URLog } from '@/classes/datas/ur-log'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import type { KyouViewEmits } from './kyou-view-emits'
import { AddURLogRequest } from '@/classes/api/req_res/add-ur-log-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/labs/components'
import delete_gkill_cache from '@/classes/delete-gkill-cache'

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
const related_date_typed: Ref<Date> = ref(moment(urlog.value.related_time).toDate())
const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
const related_time_string: Ref<string> = ref(moment(urlog.value.related_time).format("HH:mm:ss"))

const show_related_date_menu = ref(false)
const show_related_time_menu = ref(false)

function reset(): void {
    title.value = urlog.value.title
    url.value = urlog.value.url
    related_date_typed.value = moment(urlog.value.related_time).toDate()
    related_time_string.value = moment(urlog.value.related_time).format("HH:mm:ss")
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        if (!urlog.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_urlog_is_null
            error.error_message = i18n.global.t("CLIENT_URLOG_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 日時必須入力チェック
        if (related_date_string.value === "" || related_time_string.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.urlog_related_time_is_blank
            error.error_message = i18n.global.t("URLOG_DATE_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // URL入力チェック
        if (url.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.urlog_url_is_blank
            error.error_message = i18n.global.t("URLOG_URL_IS_BLANK_MESSAGE")
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
        new_urlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
        new_urlog.create_app = "gkill"
        new_urlog.create_device = gkill_info_res.device
        new_urlog.create_time = new Date(Date.now())
        new_urlog.create_user = gkill_info_res.user_id
        new_urlog.update_app = "gkill"
        new_urlog.update_device = gkill_info_res.device
        new_urlog.update_time = new Date(Date.now())
        new_urlog.update_user = gkill_info_res.user_id

        // 追加リクエストを飛ばす
        await delete_gkill_cache(new_urlog.id)
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
    related_date_typed.value = moment().toDate()
    related_time_string.value = moment().format("HH:mm:ss")
}

function reset_related_date_time(): void {
    related_date_typed.value = moment(urlog.value.related_time).toDate()
    related_time_string.value = moment(urlog.value.related_time).format("HH:mm:ss")
}
</script>

<style lang="css" scoped></style>