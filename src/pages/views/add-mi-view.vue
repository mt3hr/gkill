<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ $t("ADD_MI_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="add_notification()" :disabled="is_requested_submit">{{
                        $t("ADD_NOTIFICATION_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>{{ $t("MI_TITLE_TITLE") }}</label>
            </v-col>
            <v-col cols="auto">
                <input class="input text" type="text" v-model="mi_title" :label="$t('MI_TITLE_TITLE')" autofocus
                    :readonly="is_requested_submit" />
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>{{ $t("MI_BOARD_NAME_TITLE") }}</label>
            </v-col>
            <v-col cols="auto">
                <span>
                    <select class="select" v-model="mi_board_name" :readonly="is_requested_submit">
                        <option class="mi_board_option" v-for="board_name, index in mi_board_names" :key="index">{{
                            board_name }}</option>
                    </select>
                </span>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="secondary" class="pt-1" @click="show_new_board_name_dialog()" icon="mdi-plus" dark
                    size="small" :disabled="is_requested_submit"></v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>{{ $t("MI_START_DATE_TIME_TITLE") }}</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="mi_estimate_start_date"
                    :label="$t('MI_START_DATE_TITLE')" :readonly="is_requested_submit" />
                <input class="input time" type="time" v-model="mi_estimate_start_time" :label="$t('MI_START_TIME_TITLE')"
                    :readonly="is_requested_submit" />
            </v-col>
            <v-col cols="auto">
                <v-btn dark color="secondary" @click="clear_estimate_start_date_time()"
                    :disabled="is_requested_submit">{{ $t("CLEAR_TITLE") }}</v-btn>
                <v-btn dark color="primary" @click="now_to_estimate_start_date_time()"
                    :disabled="is_requested_submit">{{ $t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>{{ $t("MI_END_DATE_TIME_TITLE") }}</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="mi_estimate_end_date" :label="$t('MI_END_DATE_TITLE')"
                    :readonly="is_requested_submit" />
                <input class="input time" type="time" v-model="mi_estimate_end_time" :label="$t('MI_END_TIME_TITLE')"
                    :readonly="is_requested_submit" />
            </v-col>
            <v-col cols="auto">
                <v-btn dark color="secondary" @click="clear_estimate_end_date_time()" :disabled="is_requested_submit">{{
                    $t("CLEAR_TITLE") }}</v-btn>
                <v-btn dark color="primary" @click="now_to_estimate_end_date_time()" :disabled="is_requested_submit">{{
                    $t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>{{ $t("MI_LIMIT_DATE_TIME_TITLE") }}</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="mi_limit_date" :label="$t('MI_LIMIT_DATE_TITLE')"
                    :readonly="is_requested_submit" />
                <input class="input time" type="time" v-model="mi_limit_time" :label="$t('MI_LIMIT_TIME_TITLE')"
                    :readonly="is_requested_submit" />
            </v-col>
            <v-col cols="auto">
                <v-btn dark color="secondary" @click="clear_limit_date_time()" :disabled="is_requested_submit">{{
                    $t("CLEAR_TITLE") }}</v-btn>
                <v-btn dark color="primary" @click="now_to_limit_date_time()" :disabled="is_requested_submit">{{
                    $t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row v-for="notification, index in notifications" :key="notification.id" class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-row class="pa-0 ma-0">
                    <v-col cols="auto" class="pa-0 ma-0">
                        <div>{{ $t("NOTIFICATION_TITLE") }}</div>
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
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{ $t("RESET_TITLE")
                }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{ $t("SAVE_TITLE")
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

import { i18n } from '@/i18n'

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
const mi_estimate_start_date: Ref<string> = ref(mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("YYYY-MM-DD") : "")
const mi_estimate_start_time: Ref<string> = ref(mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : "")
const mi_estimate_end_date: Ref<string> = ref(mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("YYYY-MM-DD") : "")
const mi_estimate_end_time: Ref<string> = ref(mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : "")
const mi_limit_date: Ref<string> = ref(mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("YYYY-MM-DD") : "")
const mi_limit_time: Ref<string> = ref(mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : "")

const notifications: Ref<Array<Notification>> = ref(new Array<Notification>())

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
    mi_estimate_start_date.value = ""
    mi_estimate_start_time.value = ""
}

function clear_estimate_end_date_time(): void {
    mi_estimate_end_date.value = ""
    mi_estimate_end_time.value = ""
}

function clear_limit_date_time(): void {
    mi_limit_date.value = ""
    mi_limit_time.value = ""
}

function now_to_estimate_start_date_time(): void {
    mi_estimate_start_date.value = moment().format("YYYY-MM-DD")
    mi_estimate_start_time.value = moment().format("HH:mm:ss")
}

function now_to_estimate_end_date_time(): void {
    mi_estimate_end_date.value = moment().format("YYYY-MM-DD")
    mi_estimate_end_time.value = moment().format("HH:mm:ss")
}

function now_to_limit_date_time(): void {
    mi_limit_date.value = moment().format("YYYY-MM-DD")
    mi_limit_time.value = moment().format("HH:mm:ss")
}

function reset(): void {
    mi_title.value = mi.value.title
    mi_board_name.value = props.application_config.mi_default_board
    mi_estimate_start_date.value = mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("YYYY-MM-DD") : ""
    mi_estimate_start_time.value = mi.value && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : ""
    mi_estimate_end_date.value = mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("YYYY-MM-DD") : ""
    mi_estimate_end_time.value = mi.value && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : ""
    mi_limit_date.value = mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("YYYY-MM-DD") : ""
    mi_limit_time.value = mi.value && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : ""
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
        if (mi_estimate_start_date.value === "" || mi_estimate_start_time.value === "") {//どっちも入力されていなければOK。nullとして扱う
            if ((mi_estimate_start_date.value === "" && mi_estimate_start_time.value !== "") ||
                (mi_estimate_start_date.value !== "" && mi_estimate_start_time.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
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
        if (mi_estimate_end_date.value === "" || mi_estimate_end_time.value === "") {//どっちも入力されていなければOK。nullとして扱う
            if ((mi_estimate_end_date.value === "" && mi_estimate_end_time.value !== "") ||
                (mi_estimate_end_date.value !== "" && mi_estimate_end_time.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
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
        if (mi_limit_date.value === "" || mi_limit_time.value === "") {//どっちも入力されていなければOK。nullとして扱う
            if ((mi_limit_date.value === "" && mi_limit_time.value !== "") ||
                (mi_limit_date.value !== "" && mi_limit_time.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
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
            moment(mi.value.estimate_start_time) === (moment(mi_estimate_start_date.value + " " + mi_estimate_start_time.value)) &&
            moment(mi.value.estimate_end_time) === moment(mi_estimate_end_date.value + " " + mi_estimate_end_time.value) &&
            moment(mi.value.limit_time) === (moment(mi_limit_date.value + " " + mi_limit_time.value))
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
        if (mi_estimate_start_date.value !== "" && mi_estimate_start_time.value !== "") {
            estimate_start_time = moment(mi_estimate_start_date.value + " " + mi_estimate_start_time.value).toDate()
        }
        if (mi_estimate_end_date.value !== "" && mi_estimate_end_time.value !== "") {
            estimate_end_time = moment(mi_estimate_end_date.value + " " + mi_estimate_end_time.value).toDate()
        }
        if (mi_limit_date.value !== "" && mi_limit_time.value !== "") {
            limit_time = moment(mi_limit_date.value + " " + mi_limit_time.value).toDate()
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
    if (mi_estimate_start_date.value !== "" && mi_estimate_start_time.value !== "") {
        notification.notification_time = moment(mi_estimate_start_date.value + " " + mi_estimate_start_time.value).toDate()
    }
    notifications.value.push(notification)
}

function delete_notification(index: number): void {
    notifications.value.splice(index, 1)
}

load_mi_board_names()
</script>
<style lang="css" scoped>
.select {
    border: solid 1px silver;
}

.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}

.mi_board_option {
    background-color: rgb(var(--v-theme-background));
}
</style>