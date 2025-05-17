<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t('ADD_NLOG_TITLE') }}</span>
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-if="nlog" v-model="nlog_title_value" :label="i18n.global.t('NLOG_TITLE_TITLE')" autofocus
            :readonly="is_requested_submit" />
        <v-text-field v-if="nlog" v-model="nlog_shop_value" :label="i18n.global.t('NLOG_SHOP_NAME_TITLE')"
            :readonly="is_requested_submit" />
        <v-text-field v-if="nlog" v-model="nlog_amount_value" type="number" :label="i18n.global.t('NLOG_AMOUNT_TITLE')"
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_related_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_date_string"
                                        :label="i18n.global.t('LANTANA_DATE_TITLE')" readonly v-bind="props"
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
                                        :label="i18n.global.t('LANTANA_TIME_TITLE')" min-width="120" readonly
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
                <table>
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
import type { AddNlogViewProps } from './add-nlog-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, type Ref, ref } from 'vue'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import moment from 'moment'
import { Nlog } from '@/classes/datas/nlog'
import { AddNlogRequest } from '@/classes/api/req_res/add-nlog-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/labs/components'

const is_requested_submit = ref(false)

const props = defineProps<AddNlogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const nlog: Ref<Nlog> = ref((() => {
    const nlog = new Nlog()
    nlog.related_time = new Date(Date.now())
    return nlog
})())
const nlog_title_value: Ref<string> = ref("")
const nlog_amount_value: Ref<number> = ref(0)
const nlog_shop_value: Ref<string> = ref("")

const related_date_typed: Ref<Date> = ref(moment().toDate())
const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
const related_time_string: Ref<string> = ref(moment().format("HH:mm:ss"))

const show_related_date_menu = ref(false)
const show_related_time_menu = ref(false)

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        if (!nlog.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_nlog_is_null
            error.error_message = i18n.global.t("CLIENT_NLOG_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 日時必須入力チェック
        if (related_date_string.value === "" || related_time_string.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.nlog_related_time_is_blank
            error.error_message = i18n.global.t("NLOG_DATE_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 金額入力チェック
        if (Number.isNaN(nlog_amount_value.value) || nlog_amount_value.value.toString() === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.nlog_amount_is_blank
            error.error_message = i18n.global.t("NLOG_AMOUNT_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 店名入力チェック
        if (nlog_shop_value.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.nlog_shop_name_is_blank
            error.error_message = i18n.global.t("NLOG_SHOP_NAME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // タイトル入力チェック
        if (nlog_title_value.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.nlog_title_is_blank
            error.error_message = i18n.global.t("NLOG_TITLE_IS_BLANK_MESSAGE")
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

        // 更新後Nlog情報を用意する
        const new_nlog = await nlog.value.clone()
        new_nlog.id = props.gkill_api.generate_uuid()
        new_nlog.amount = nlog_amount_value.value
        new_nlog.shop = nlog_shop_value.value
        new_nlog.title = nlog_title_value.value
        new_nlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
        new_nlog.create_app = "gkill"
        new_nlog.create_device = gkill_info_res.device
        new_nlog.create_time = new Date(Date.now())
        new_nlog.create_user = gkill_info_res.user_id
        new_nlog.update_app = "gkill"
        new_nlog.update_device = gkill_info_res.device
        new_nlog.update_time = new Date(Date.now())
        new_nlog.update_user = gkill_info_res.user_id

        // 追加リクエストを飛ばす
        const req = new AddNlogRequest()
        req.nlog = new_nlog
        const res = await props.gkill_api.add_nlog(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits('registered_kyou', res.added_nlog_kyou)
        emits('requested_reload_list')
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}

function reset_related_date_time(): void {
    related_date_typed.value = moment(nlog.value.related_time).toDate()
    related_time_string.value = moment(nlog.value.related_time).format("HH:mm:ss")
}

function now_to_related_date_time(): void {
    related_date_typed.value = moment().toDate()
    related_time_string.value = moment().format("HH:mm:ss")
}

function reset(): void {
    nlog_title_value.value = (nlog.value ? nlog.value.title : "")
    nlog_amount_value.value = (nlog.value ? nlog.value.amount : 0)
    nlog_shop_value.value = (nlog.value ? nlog.value.shop : "")
    related_date_typed.value = (moment().toDate())
    related_time_string.value = (moment().format("HH:mm:ss"))
}
</script>

<style lang="css" scoped></style>