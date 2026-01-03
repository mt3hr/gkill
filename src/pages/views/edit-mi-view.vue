<template>
    <v-card v-if="cloned_kyou.typed_mi" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("EDIT_MI_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field class="input text" type="text" v-model="mi_title" :label="i18n.global.t('MI_TITLE_TITLE')"
            autofocus :readonly="is_requested_submit" />
        <table>
            <tr>
                <td>
                    <v-select class="select" v-model="mi_board_name" :items="mi_board_names"
                        :readonly="is_requested_submit" :label="i18n.global.t('MI_BOARD_NAME_TITLE')" />
                </td>
                <td>
                    <v-btn color="primary" @click="show_new_board_name_dialog()" icon="mdi-plus" dark size="small"
                        :disabled="is_requested_submit"></v-btn>
                </td>
            </tr>
        </table>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_start_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="mi_estimate_start_date_string"
                                        :label="i18n.global.t('MI_START_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="mi_estimate_start_date_typed"
                                    @update:model-value="show_start_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_start_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="mi_estimate_start_time_string"
                                        :label="i18n.global.t('MI_START_TIME_TITLE')" min-width="120" readonly
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="mi_estimate_start_time_string" format="24hr"
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
                            <v-btn dark color="secondary" @click="clear_estimate_start_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="secondary" @click="reset_estimate_start_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_estimate_start_date_time()"
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
                                    <v-text-field v-model="mi_estimate_end_date_string"
                                        :label="i18n.global.t('MI_END_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="mi_estimate_end_date_typed"
                                    @update:model-value="show_end_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_end_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="mi_estimate_end_time_string"
                                        :label="i18n.global.t('MI_END_TIME_TITLE')" min-width="120" readonly
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="mi_estimate_end_time_string" format="24hr"
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
                            <v-btn dark color="secondary" @click="clear_estimate_end_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="secondary" @click="reset_estimate_end_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_estimate_end_date_time()"
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
                            <v-menu v-model="show_limit_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="mi_limit_date_string"
                                        :label="i18n.global.t('MI_LIMIT_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="mi_limit_date_typed"
                                    @update:model-value="show_limit_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_limit_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="mi_limit_time_string"
                                        :label="i18n.global.t('MI_LIMIT_TIME_TITLE')" min-width="120" readonly
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="mi_limit_time_string" format="24hr"
                                    @update:minute="show_limit_time_menu = false" />
                            </v-menu>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="pt-2">
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="clear_limit_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="secondary" @click="reset_limit_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_limit_date_time()"
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
                :show_timeis_plaing_end_button="false" :highlight_targets="highlight_targets" :is_image_view="false"
                :kyou="kyou" :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_mi_plaing_end_button="true" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_attached_timeis="true"
                :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
                :show_update_time="false" :show_related_time="true"
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
        <NewBoardNameDialog v-if="kyou.typed_mi" :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @setted_new_board_name="(board_name: string) => update_board_name(board_name)"
            ref="new_board_name_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
import type { EditMiViewProps } from './edit-mi-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue'
import moment from 'moment'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { GkillError } from '@/classes/api/gkill-error'

import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

const new_board_name_dialog = ref<InstanceType<typeof NewBoardNameDialog> | null>(null);

const is_requested_submit = ref(false)

const props = defineProps<EditMiViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const show_kyou: Ref<boolean> = ref(false)
const mi_board_names: Ref<Array<string>> = ref(props.application_config.mi_default_board !== "" ? [props.application_config.mi_default_board] : [])

const mi_title: Ref<string> = ref(cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.title : "")
const mi_board_name: Ref<string> = ref(cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.board_name : "")
const mi_estimate_start_date_typed: Ref<Date | null> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).toDate() : null)
const mi_estimate_start_date_string: Ref<string> = computed(() => mi_estimate_start_date_typed.value ? moment(mi_estimate_start_date_typed.value).format("YYYY-MM-DD") : "")
const mi_estimate_start_time_string: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : "")
const mi_estimate_end_date_typed: Ref<Date | null> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).toDate() : null)
const mi_estimate_end_date_string: Ref<string> = computed(() => mi_estimate_end_date_typed.value ? moment(mi_estimate_end_date_typed.value).format("YYYY-MM-DD") : "")
const mi_estimate_end_time_string: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : "")
const mi_limit_date_typed: Ref<Date | null> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).toDate() : null)
const mi_limit_date_string: Ref<string> = computed(() => mi_limit_date_typed.value ? moment(mi_limit_date_typed.value).format("YYYY-MM-DD") : "")
const mi_limit_time_string: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : "")

const show_start_date_menu = ref(false)
const show_start_time_menu = ref(false)
const show_end_date_menu = ref(false)
const show_end_time_menu = ref(false)
const show_limit_date_menu = ref(false)
const show_limit_time_menu = ref(false)

watch(() => props.kyou, () => load())
load()

async function load(): Promise<void> {
    cloned_kyou.value = props.kyou.clone()
    await cloned_kyou.value.reload(false, true)
    await cloned_kyou.value.load_typed_datas()
    cloned_kyou.value.load_all()
    mi_title.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.title : ""
    mi_board_name.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.board_name : ""
    mi_estimate_start_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).toDate() : null
    mi_estimate_start_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : ""
    mi_estimate_end_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).toDate() : null
    mi_estimate_end_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : ""
    mi_limit_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).toDate() : null
    mi_limit_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : ""
}

async function load_mi_board_names(): Promise<void> {
    const req = new GetMiBoardRequest()

    const res = await props.gkill_api.get_mi_board_list(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        // emits('received_messages', res.messages)
    }

    let is_contain_default_board = false
    res.boards.forEach((board_name) => {
        if (board_name === props.application_config.mi_default_board) {
            is_contain_default_board = true
        }
    })
    if (!is_contain_default_board) {
        res.boards.push(props.application_config.mi_default_board)
    }

    mi_board_names.value = res.boards
}

function update_board_name(board_name: string): void {
    mi_board_names.value.push(board_name)
    mi_board_name.value = board_name
}

function show_new_board_name_dialog(): void {
    new_board_name_dialog.value?.show()
}

function clear_estimate_start_date_time(): void {
    mi_estimate_start_date_typed.value = null
    mi_estimate_start_time_string.value = ""
}

function clear_estimate_end_date_time(): void {
    mi_estimate_end_date_typed.value = null
    mi_estimate_end_time_string.value = ""
}

function clear_limit_date_time(): void {
    mi_limit_date_typed.value = null
    mi_limit_time_string.value = ""
}

function reset_estimate_start_date_time(): void {
    mi_estimate_start_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).toDate() : null
    mi_estimate_start_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : ""
}

function reset_estimate_end_date_time(): void {
    mi_estimate_end_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).toDate() : null
    mi_estimate_end_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : ""
}

function reset_limit_date_time(): void {
    mi_limit_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).toDate() : null
    mi_limit_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : ""
}

function now_to_estimate_start_date_time(): void {
    mi_estimate_start_date_typed.value = moment().toDate()
    mi_estimate_start_time_string.value = moment().format("HH:mm:ss")
}

function now_to_estimate_end_date_time(): void {
    mi_estimate_end_date_typed.value = moment().toDate()
    mi_estimate_end_time_string.value = moment().format("HH:mm:ss")
}

function now_to_limit_date_time(): void {
    mi_limit_date_typed.value = moment().toDate()
    mi_limit_time_string.value = moment().format("HH:mm:ss")
}

function reset(): void {
    mi_title.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.title : ""
    mi_board_name.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.board_name : ""
    mi_estimate_start_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).toDate() : null
    mi_estimate_start_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : ""
    mi_estimate_end_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).toDate() : null
    mi_estimate_end_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : ""
    mi_limit_date_typed.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).toDate() : null
    mi_limit_time_string.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : ""
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        cloned_kyou.value.abort_controller.abort()
        cloned_kyou.value.abort_controller = new AbortController()

        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        const mi = cloned_kyou.value.typed_mi
        if (!mi) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_mi_is_null
            error.error_message = i18n.global.t("CLIENT_MI_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // タイトルの入力チェック
        if (mi_title.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.mi_title_is_blank
            error.error_message = i18n.global.t("MI_TITLE_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 開始日時 片方だけ入力されていたらエラーチェック
        if (mi_estimate_start_date_string.value === "" || mi_estimate_start_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
            if ((mi_estimate_start_date_string.value === "" && mi_estimate_start_time_string.value !== "") ||
                (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                const error = new GkillError()
                error.error_code = GkillErrorCodes.mi_estimate_start_time_is_blank
                error.error_message = i18n.global.t("MI_START_DATE_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }
        }

        // 終了日時 片方だけ入力されていたらエラーチェック
        if (mi_estimate_end_date_string.value === "" || mi_estimate_end_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
            if ((mi_estimate_end_date_string.value === "" && mi_estimate_end_time_string.value !== "") ||
                (mi_estimate_end_date_string.value !== "" && mi_estimate_end_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                const error = new GkillError()
                error.error_code = GkillErrorCodes.mi_estimate_end_time_is_blank
                error.error_message = i18n.global.t("MI_END_DATE_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }
        }

        // 期限日時 片方だけ入力されていたらエラーチェック
        if (mi_limit_date_string.value === "" || mi_limit_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
            if ((mi_limit_date_string.value === "" && mi_limit_time_string.value !== "") ||
                (mi_limit_date_string.value !== "" && mi_limit_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                const error = new GkillError()
                error.error_code = GkillErrorCodes.mi_limit_time_is_blank
                error.error_message = i18n.global.t("MI_LIMIT_DATE_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }
        }

        // 更新がなかったらエラーメッセージを出力する
        if (mi.title === mi_title.value &&
            mi.board_name === mi_board_name.value &&
            (moment(mi.estimate_start_time).toDate().getTime() === moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).toDate().getTime() || (mi.estimate_start_time == null && mi_estimate_start_date_string.value === "" && mi_estimate_start_time_string.value === "")) &&
            (moment(mi.estimate_end_time).toDate().getTime() === moment(mi_estimate_end_date_string.value + " " + mi_estimate_end_time_string.value).toDate().getTime() || (mi.estimate_end_time == null && mi_estimate_end_date_string.value === "" && mi_estimate_end_time_string.value === "")) &&
            (moment(mi.limit_time).toDate().getTime() === moment(mi_limit_date_string.value + " " + mi_limit_time_string.value).toDate().getTime() || (mi.limit_time == null && mi_limit_date_string.value === "" && mi_limit_time_string.value === ""))) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.mi_is_no_update
            error.error_message = i18n.global.t("MI_IS_NO_UPDATE_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新後Mi情報を用意する
        let estimate_start_time: Date | null = null
        let estimate_end_time: Date | null = null
        let limit_time: Date | null = null
        if (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value !== "") {
            estimate_start_time = moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).toDate()
        }
        if (mi_estimate_end_date_string.value !== "" && mi_estimate_end_time_string.value !== "") {
            estimate_end_time = moment(mi_estimate_end_date_string.value + " " + mi_estimate_end_time_string.value).toDate()
        }
        if (mi_limit_date_string.value !== "" && mi_limit_time_string.value !== "") {
            limit_time = moment(mi_limit_date_string.value + " " + mi_limit_time_string.value).toDate()
        }
        const updated_mi = await mi.clone()
        updated_mi.title = mi_title.value
        updated_mi.board_name = mi_board_name.value
        updated_mi.estimate_start_time = estimate_start_time
        updated_mi.estimate_end_time = estimate_end_time
        updated_mi.limit_time = limit_time
        updated_mi.update_app = "gkill"
        updated_mi.update_device = props.application_config.device
        updated_mi.update_time = new Date(Date.now())
        updated_mi.update_user = props.application_config.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_mi.id)
        const req = new UpdateMiRequest()
        req.mi = updated_mi

        const res = await props.gkill_api.update_mi(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits("updated_kyou", res.updated_mi_kyou)
        emits('requested_reload_kyou', props.kyou)
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}

load_mi_board_names()
</script>

<style lang="css" scoped></style>