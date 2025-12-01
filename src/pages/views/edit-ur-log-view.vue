<template>
    <v-card v-if="cloned_kyou.typed_urlog" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("EDIT_URLOG_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
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
            <tr>
                <td>
                    <v-checkbox v-model="re_get_urlog_content" :label="i18n.global.t('URLOG_REGET_TITLE')" hide-details
                        color="primary" />
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
                                    @update:minute="show_related_time_menu = false" />
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
            <KyouView :application_config="application_config" :gkill_api="gkill_api" :show_timeis_elapsed_time="true"
                :show_timeis_plaing_end_button="true" :highlight_targets="highlight_targets" :is_image_view="false"
                :kyou="kyou" :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_urlog_plaing_end_button="true" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_attached_timeis="true"
                :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
                @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)" :show_update_time="false" :show_related_time="true"
                @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
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
import type { EditURLogViewProps } from './edit-ur-log-view-props'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateURLogRequest } from '@/classes/api/req_res/update-ur-log-request'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

const is_requested_submit = ref(false)

const props = defineProps<EditURLogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const title: Ref<string> = ref(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.title : "")
const url: Ref<string> = ref(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.url : "")
const related_date_typed: Ref<Date> = ref(moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").toDate())
const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
const related_time_string: Ref<string> = ref(moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").format("HH:mm:ss"))
const re_get_urlog_content: Ref<boolean> = ref(true)

const show_kyou: Ref<boolean> = ref(false)

const show_related_date_menu = ref(false)
const show_related_time_menu = ref(false)

watch(() => props.kyou, () => load())
load()

async function load(): Promise<void> {
    cloned_kyou.value = props.kyou.clone()
    await cloned_kyou.value.load_typed_datas()
    cloned_kyou.value.load_all()
    title.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.title : ""
    url.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.url : ""
    related_date_typed.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").toDate()
    related_time_string.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").format("HH:mm:ss")
}

function reset(): void {
    title.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.title : ""
    url.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.url : ""
    related_date_typed.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").toDate()
    related_time_string.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").format("HH:mm:ss")
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        cloned_kyou.value.abort_controller.abort()
        cloned_kyou.value.abort_controller = new AbortController()

        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        const urlog = cloned_kyou.value.typed_urlog
        if (!urlog) {
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

        // 更新がなかったらエラーメッセージを出力する
        if (urlog.title === title.value &&
            urlog.url === url.value &&
            moment(urlog.related_time).toDate().getTime() === moment(related_date_string.value + " " + related_time_string.value).toDate().getTime() &&
            moment(urlog.related_time).toDate().getTime() === moment(related_date_string.value + " " + related_time_string.value).toDate().getTime()) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.urlog_is_no_update
            error.error_message = i18n.global.t("URLOG_IS_NO_UPDATE_MESSAGE")
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
        const updated_urlog = await urlog.clone()
        updated_urlog.title = title.value
        updated_urlog.url = url.value
        updated_urlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
        updated_urlog.update_app = "gkill"
        updated_urlog.update_device = gkill_info_res.device
        updated_urlog.update_time = new Date(Date.now())
        updated_urlog.update_user = gkill_info_res.user_id

        // 再取得の場合、URLとタイトル以外をブランクにする
        if (re_get_urlog_content.value) {
            updated_urlog.description = ""
            updated_urlog.favicon_image = ""
            updated_urlog.thumbnail_image = ""
        }

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_urlog.id)
        const req = new UpdateURLogRequest()
        req.urlog = updated_urlog

        const res = await props.gkill_api.update_urlog(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits("updated_kyou", res.updated_urlog_kyou)
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
    related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
    related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
}
</script>

<style lang="css" scoped></style>