<template>
    <v-card v-if="cloned_kyou.typed_timeis" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("EDIT_TIMEIS_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field class="input text" type="text" v-model="timeis_title" :label="i18n.global.t('TIMEIS_TITLE_TITLE')"
            autofocus :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
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
                                    @update:minute="show_start_time_menu = false" />
                            </v-menu>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="pt-2">
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
            <v-col cols="auto" class="pa-0 ma-0">
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
                                    @update:minute="show_end_time_menu = false" />
                            </v-menu>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="pt-2">
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="reset_end_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="secondary" @click="clear_end_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
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
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="true" :show_attached_timeis="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_update_time="false"
                :show_related_time="true" @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
                :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { EditTimeIsViewProps } from './edit-time-is-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import KyouView from './kyou-view.vue'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'

import { UpdateTimeisRequest } from '@/classes/api/req_res/update-timeis-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

const is_requested_submit = ref(false)

const props = defineProps<EditTimeIsViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const timeis_title: Ref<string> = ref(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : "")
const timeis_start_date_typed: Ref<Date> = ref(cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis?.start_time).toDate() : new Date(0))
const timeis_start_date_string: Ref<string> = computed(() => moment(timeis_start_date_typed.value).format("YYYY-MM-DD"))
const timeis_start_time_string: Ref<string> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.start_time).format("HH:mm:ss") : "")
const timeis_end_date_typed: Ref<Date | null> = ref(cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).toDate() : null)
const timeis_end_date_string: Ref<string> = computed(() => timeis_end_date_typed.value ? moment(timeis_end_date_typed.value).format("YYYY-MM-DD") : "")
const timeis_end_time_string: Ref<string> = ref(timeis_end_date_typed.value ? moment(timeis_end_date_typed.value).format("HH:mm:ss") : "")

const show_kyou: Ref<boolean> = ref(false)
const show_start_date_menu = ref(false)
const show_start_time_menu = ref(false)
const show_end_date_menu = ref(false)
const show_end_time_menu = ref(false)

watch(() => props.kyou, () => load())
load()

async function load(): Promise<void> {
    cloned_kyou.value = props.kyou.clone()
    await cloned_kyou.value.reload(false, true)
    await cloned_kyou.value.load_typed_datas()
    cloned_kyou.value.load_all()
    timeis_title.value = cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : ""
    timeis_start_date_typed.value = moment(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.start_time : "").toDate()
    timeis_start_time_string.value = moment(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.start_time : "").format("HH:mm:ss")
    timeis_end_date_typed.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).toDate() : null
    timeis_end_time_string.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).format("HH:mm:ss") : ""
}

function reset(): void {
    timeis_title.value = cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : ""
    reset_start_date_time()
    reset_end_date_time()
}

function reset_start_date_time(): void {
    timeis_start_date_typed.value = cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis.start_time).toDate() : new Date(0)
    timeis_start_time_string.value = cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis.start_time).format("HH:mm:ss") : ""
}

function reset_end_date_time(): void {
    timeis_end_date_typed.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).toDate() : null
    timeis_end_time_string.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).format("HH:mm:ss") : ""
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
        cloned_kyou.value.abort_controller.abort()
        cloned_kyou.value.abort_controller = new AbortController()

        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        const timeis = cloned_kyou.value.typed_timeis
        if (!timeis) {
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

        // 更新がなかったらエラーメッセージを出力する
        if (timeis.title === timeis_title.value &&
            (moment(timeis.start_time).toDate().getTime() === moment(timeis_start_date_string.value + " " + timeis_start_time_string.value).toDate().getTime()) &&
            (moment(timeis.end_time).toDate().getTime() === moment(timeis_end_date_string.value + " " + timeis_end_time_string.value).toDate().getTime() || (timeis.end_time === null && timeis_end_date_string.value === "" && timeis_end_time_string.value === ""))) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.timeis_is_no_update
            error.error_message = i18n.global.t("TIMEIS_IS_NO_UPDATE_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新後TimeIs情報を用意する
        let end_time: Date | null = null
        if (timeis_end_date_string.value !== "" && timeis_end_time_string.value !== "") {
            end_time = moment(timeis_end_date_string.value + " " + timeis_end_time_string.value).toDate()
        }
        const updated_timeis = await timeis.clone()
        updated_timeis.title = timeis_title.value
        updated_timeis.start_time = moment(timeis_start_date_string.value + " " + timeis_start_time_string.value).toDate()
        updated_timeis.end_time = end_time
        updated_timeis.update_app = "gkill"
        updated_timeis.update_device = props.application_config.device
        updated_timeis.update_time = new Date(Date.now())
        updated_timeis.update_user = props.application_config.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_timeis.id)
        const req = new UpdateTimeisRequest()
        req.timeis = updated_timeis
        req.want_response_kyou = true
        const res = await props.gkill_api.update_timeis(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits("updated_kyou", res.updated_kyou!)
        emits('requested_reload_kyou', props.kyou)
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}

function clear_end_date_time(): void {
    timeis_end_date_typed.value = null
    timeis_end_time_string.value = ""
}


</script>

<style lang="css" scoped></style>