<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_MI_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="add_notification()" :disabled="is_requested_submit">{{
                        i18n.global.t("ADD_NOTIFICATION_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <table>
            <tr>
                <td>
                    <v-text-field class="input text" type="text" v-model="mi_title"
                        :label="i18n.global.t('MI_TITLE_TITLE')" autofocus :readonly="is_requested_submit" />
                </td>
            </tr>
            <tr>
                <td>
                    <v-select class="select" v-model="mi_board_name" :items="mi_board_names"
                        :readonly="is_requested_submit" />
                </td>
                <td>
                    <v-btn color="secondary" class="pt-1" @click="show_new_board_name_dialog()" icon="mdi-plus" dark
                        size="small" :disabled="is_requested_submit"></v-btn>
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
                                    @update:model-value="show_start_time_menu = false" />
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
                                    @update:model-value="show_end_time_menu = false" />
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
                                    @update:model-value="show_limit_time_menu = false" />
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
                            <v-btn dark color="primary" @click="now_to_limit_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>
        <v-row v-for="notification, index in notifications" :key="notification.id" class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-row class="pa-0 ma-0">
                    <v-col cols="auto" class="pa-0 ma-0">
                        <div>{{ i18n.global.t("NOTIFICATION_TITLE") }}</div>
                    </v-col>
                    <v-spacer />
                    <v-col cols="auto" class="pa-0 ma-0">
                        <v-btn class="rounded-sm mx-auto" icon @click.prevent="delete_notification(index)"
                            :disabled="is_requested_submit">
                            <v-icon>mdi-close</v-icon>
                        </v-btn>
                    </v-col>
                </v-row>
                <v-row class="pa-0 ma-0">
                    <v-col cols="auto" class="pa-0 ma-0">
                        <AddNotificationForAddMiView :application_config="application_config" :gkill_api="gkill_api"
                            :enable_context_menu="false" :enable_dialog="true" :highlight_targets="[]" :kyou="kyou"
                            :last_added_tag="''" :default_notification="notification" ref="add_notification_views"
                            @received_errors="(errors) => emits('received_errors', errors)"
                            @received_messages="(messages) => emits('received_messages', messages)" />
                    </v-col>
                </v-row>
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
        <NewBoardNameDialog v-if="mi" :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @setted_new_board_name="(board_name: string) => update_board_name(board_name)"
            ref="new_board_name_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import AddNotificationForAddMiView from './add-notification-for-add-mi-view.vue'
import type { AddMiViewProps } from './add-mi-view-props'
import { computed, type Ref, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import { Mi } from '@/classes/datas/mi'
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue'
import moment from 'moment'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { AddMiRequest } from '@/classes/api/req_res/add-mi-request'
import { Kyou } from '@/classes/datas/kyou'
import { Notification } from '@/classes/datas/notification'
import { AddNotificationRequest } from '@/classes/api/req_res/add-notification-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/labs/components'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

const new_board_name_dialog = ref<InstanceType<typeof NewBoardNameDialog> | null>(null);
const add_notification_views = ref<any>(null);

const is_requested_submit = ref(false)

const props = defineProps<AddMiViewProps>()
const emits = defineEmits<KyouViewEmits>()

const id: Ref<string> = ref(props.gkill_api.generate_uuid())
const kyou: Ref<Kyou> = computed(() => {
    const kyou = new Kyou()
    kyou.id = id.value
    return kyou
})
const mi: Ref<Mi> = ref((() => {
    const mi = new Mi()
    mi.id = id.value
    return mi
})())
const mi_board_names: Ref<Array<string>> = ref(props.application_config.mi_default_board !== "" ? [props.application_config.mi_default_board] : [])

const mi_title: Ref<string> = ref(mi.value ? mi.value.title : "")
const mi_board_name: Ref<string> = ref(props.application_config.mi_default_board !== "" ? props.application_config.mi_default_board : "Inbox")
const mi_estimate_start_date_typed: Ref<Date | null> = ref(mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).toDate() : null)
const mi_estimate_start_date_string: Ref<string> = computed(() => mi_estimate_start_date_typed.value ? moment(mi_estimate_start_date_typed.value).format("YYYY-MM-DD") : "")
const mi_estimate_start_time_string: Ref<string> = ref(mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : "")
const mi_estimate_end_date_typed: Ref<Date | null> = ref(mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).toDate() : null)
const mi_estimate_end_date_string: Ref<string> = computed(() => mi_estimate_end_date_typed.value ? moment(mi_estimate_end_date_typed.value).format("YYYY-MM-DD") : "")
const mi_estimate_end_time_string: Ref<string> = ref(mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : "")
const mi_limit_date_typed: Ref<Date | null> = ref(mi.value && mi.value.limit_time ? moment(mi.value.limit_time).toDate() : null)
const mi_limit_date_string: Ref<string> = computed(() => mi_limit_date_typed.value ? moment(mi_limit_date_typed.value).format("YYYY-MM-DD") : "")
const mi_limit_time_string: Ref<string> = ref(mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : "")

const notifications: Ref<Array<Notification>> = ref(new Array<Notification>())
const show_start_date_menu = ref(false)
const show_start_time_menu = ref(false)
const show_end_date_menu = ref(false)
const show_end_time_menu = ref(false)
const show_limit_date_menu = ref(false)
const show_limit_time_menu = ref(false)

watch(() => props.application_config, () => load_mi_board_names)
load_mi_board_names()

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
    mi_title.value = mi.value.title
    mi_board_name.value = props.application_config.mi_default_board
    mi_estimate_start_date_typed.value = mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).toDate() : null
    mi_estimate_start_time_string.value = mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : ""
    mi_estimate_end_date_typed.value = mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).toDate() : null
    mi_estimate_end_time_string.value = mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : ""
    mi_limit_date_typed.value = mi.value && mi.value.limit_time ? moment(mi.value.limit_time).toDate() : null
    mi_limit_time_string.value = mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : ""
    notifications.value.splice(0)
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // Notification チェック
        // おかしかったらnullが戻ってくるので中断する
        const notifications = new Array<Notification>()
        if (add_notification_views.value) {
            for (let i = 0; i < add_notification_views.value.length; i++) {
                const notification = await add_notification_views.value[i].get_notification()
                if (!notification) {
                    return
                }
                notifications.push(notification)
            }
        }

        // Mi チェック
        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        if (!mi.value) {
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
        if (mi.value.title === mi_title.value &&
            moment(mi.value.estimate_start_time) === (moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value)) &&
            moment(mi.value.estimate_end_time) === moment(mi_estimate_end_date_string.value + " " + mi_estimate_end_time_string.value) &&
            moment(mi.value.limit_time) === (moment(mi_limit_date_string.value + " " + mi_limit_time_string.value))
        ) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.mi_is_no_update
            error.error_message = i18n.global.t("MI_IS_NO_UPDATE_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // Mi 追加

        // UserIDやDevice情報を取得する
        const get_gkill_req = new GetGkillInfoRequest()
        const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
        if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
            emits('received_errors', gkill_info_res.errors)
            return
        }

        // 作成するMi情報を用意する
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
        const new_mi = await mi.value.clone()
        new_mi.id = mi.value.id
        new_mi.title = mi_title.value
        new_mi.board_name = mi_board_name.value
        new_mi.estimate_start_time = estimate_start_time
        new_mi.estimate_end_time = estimate_end_time
        new_mi.limit_time = limit_time
        new_mi.create_app = "gkill"
        new_mi.create_device = gkill_info_res.device
        new_mi.create_time = new Date(Date.now())
        new_mi.create_user = gkill_info_res.user_id
        new_mi.update_app = "gkill"
        new_mi.update_device = gkill_info_res.device
        new_mi.update_time = new Date(Date.now())
        new_mi.update_user = gkill_info_res.user_id

        // 追加リクエストを飛ばす
        await delete_gkill_kyou_cache(new_mi.id)
        const req = new AddMiRequest()
        req.mi = new_mi
        const res = await props.gkill_api.add_mi(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }

        // Notification 追加
        for (let i = 0; i < notifications.length; i++) {
            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(notifications[i].id)
            const req = new AddNotificationRequest()
            req.notification = notifications[i]
            const res = await props.gkill_api.add_notification(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
        }



        emits("registered_kyou", res.added_mi_kyou)
        emits('requested_reload_list')
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}

function add_notification(): void {
    const notification = new Notification()
    notification.id = props.gkill_api.generate_uuid()
    notification.target_id = id.value
    notification.content = mi_title.value
    notification.notification_time = new Date(0)
    if (mi_estimate_start_date_string.value !== "" && mi_estimate_start_time_string.value !== "") {
        notification.notification_time = moment(mi_estimate_start_date_string.value + " " + mi_estimate_start_time_string.value).toDate()
    }
    notifications.value.push(notification)
}

function delete_notification(index: number): void {
    notifications.value.splice(index, 1)
}

load_mi_board_names()
</script>
<style lang="css" scoped></style>