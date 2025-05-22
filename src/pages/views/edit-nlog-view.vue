<template>
    <v-card v-if="cloned_kyou.typed_nlog" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("EDIT_NLOG_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-if="kyou.typed_nlog" v-model="nlog_title_value" :label="i18n.global.t('NLOG_TITLE_TITLE')"
            autofocus :readonly="is_requested_submit" />
        <v-text-field v-if="kyou.typed_nlog" v-model="nlog_shop_value" :label="i18n.global.t('NLOG_SHOP_NAME_TITLE')"
            :readonly="is_requested_submit" />
        <v-text-field v-if="kyou.typed_nlog" v-model="nlog_amount_value" type="number"
            :label="i18n.global.t('NLOG_AMOUNT_TITLE')" :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_related_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_date_string"
                                        :label="i18n.global.t('NLOG_DATE_TITLE')" readonly v-bind="props"
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
                                        :label="i18n.global.t('NLOG_TIME_TITLE')" readonly min-width="120"
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

        <v-card v-if="show_kyou">
            <KyouView v-if="kyou.typed_nlog" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="true" :show_attached_timeis="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_related_time="true"
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
import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import moment from 'moment'
import { UpdateNlogRequest } from '@/classes/api/req_res/update-nlog-request'
import type { EditNlogViewProps } from './edit-nlog-view-props'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/labs/components'
import delete_gkill_cache from '@/classes/delete-gkill-cache'

const is_requested_submit = ref(false)

const props = defineProps<EditNlogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const nlog_title_value: Ref<string> = ref(props.kyou.typed_nlog ? props.kyou.typed_nlog.title : "")
const nlog_amount_value: Ref<number> = ref(props.kyou.typed_nlog ? props.kyou.typed_nlog.amount : 0)
const nlog_shop_value: Ref<string> = ref(props.kyou.typed_nlog ? props.kyou.typed_nlog.shop : "")

const related_date_typed: Ref<Date> = ref(moment(props.kyou.related_time).toDate())
const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
const related_time_string: Ref<string> = ref(moment(props.kyou.related_time).format("HH:mm:ss"))
const show_kyou: Ref<boolean> = ref(false)

const show_related_date_menu = ref(false)
const show_related_time_menu = ref(false)

watch(() => props.kyou, () => load())
load()

async function load(): Promise<void> {
    cloned_kyou.value = props.kyou.clone()
    await cloned_kyou.value.load_typed_datas()
    cloned_kyou.value.load_all()
    nlog_title_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.title : ""
    nlog_amount_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.amount : 0
    nlog_shop_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.shop : ""
    related_date_typed.value = moment(props.kyou.related_time).toDate()
    related_time_string.value = moment(props.kyou.related_time).format("HH:mm:ss")
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        cloned_kyou.value.abort_controller.abort()
        cloned_kyou.value.abort_controller = new AbortController()

        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        const nlog = props.kyou.typed_nlog
        if (!nlog) {
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
        if (Number.isNaN(nlog_amount_value.value)) {
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

        // 更新がなかったらエラーメッセージを出力する
        if (nlog_amount_value.value === nlog.amount &&
            nlog_shop_value.value === nlog.shop &&
            nlog_title_value.value === nlog.title &&
            moment(related_date_string.value + " " + related_time_string.value).toDate().getTime() === moment(nlog.related_time).toDate().getTime()) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.nlog_is_no_update
            error.error_message = i18n.global.t("NLOG_IS_NO_UPDATE_MESSAGE")
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

        // 更新後Kmemo情報を用意する
        const updated_nlog = await nlog.clone()
        updated_nlog.amount = nlog_amount_value.value
        updated_nlog.shop = nlog_shop_value.value
        updated_nlog.title = nlog_title_value.value
        updated_nlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
        updated_nlog.update_app = "gkill"
        updated_nlog.update_device = gkill_info_res.device
        updated_nlog.update_time = new Date(Date.now())
        updated_nlog.update_user = gkill_info_res.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_cache(updated_nlog.id)
        const req = new UpdateNlogRequest()
        req.nlog = updated_nlog

        const res = await props.gkill_api.update_nlog(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits('updated_kyou', res.updated_nlog_kyou)
        emits('requested_reload_kyou', props.kyou)
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
    related_date_typed.value = moment(props.kyou.related_time).toDate()
    related_time_string.value = moment(props.kyou.related_time).format("HH:mm:ss")
}

function reset(): void {
    nlog_title_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.title : ""
    nlog_amount_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.amount : 0
    nlog_shop_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.shop : ""
    related_date_typed.value = moment(props.kyou.related_time).toDate()
    related_time_string.value = moment(props.kyou.related_time).format("HH:mm:ss")

}
</script>

<style lang="css" scoped></style>