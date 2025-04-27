<template>
    <v-card v-if="cloned_kyou.typed_mi" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ $t("EDIT_MI_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="$t('SHOW_TARGET_KYOU')" hide-details color="primary" />
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
                <label>{{ $t('MI_BOARD_NAME_TITLE') }}</label>
            </v-col>
            <v-col cols="auto">
                <span>
                    <select class="select" v-model="mi_board_name">
                        <option class="mi_board_option" v-for="board_name in mi_board_names" :key="board_name">{{
                            board_name }}</option>
                    </select>
                </span>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn class="pt-1" @click="show_new_board_name_dialog()" icon="mdi-plus" dark size="small"
                    :disabled="is_requested_submit"></v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>{{ $t("MI_START_DATE_TIME_TITLE") }}</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="mi_estimate_start_date"
                    :label="$t('MI_START_DATE_TITLE')" :readonly="is_requested_submit" />
                <input class="input time" type="time" v-model="mi_estimate_start_time"
                    :label="$t('MI_START_TIME_TITLE')" :readonly="is_requested_submit" />
            </v-col>
            <v-col cols="auto">
                <v-btn dark color="secondary" @click="clear_estimate_start_date_time()"
                    :disabled="is_requested_submit">{{ $t("CLEAR_TITLE") }}</v-btn>
                <v-btn dark color="secondary" @click="reset_estimate_start_date_time()"
                    :disabled="is_requested_submit">{{ $t("RESET_TITLE") }}</v-btn>
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
                <v-btn dark color="secondary" @click="reset_estimate_end_date_time()" :disabled="is_requested_submit">{{
                    $t("RESET_TITLE") }}</v-btn>
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
                <v-btn dark color="secondary" @click="clear_limit_date_time()"
                    :disabled="is_requested_submit">{{ $t("CLEAR_TITLE") }}</v-btn>
                <v-btn dark color="secondary" @click="reset_limit_date_time()"
                    :disabled="is_requested_submit">{{ $t("RESET_TITLE") }}</v-btn>
                <v-btn dark color="primary" @click="now_to_limit_date_time()"
                    :disabled="is_requested_submit">{{ $t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{ $t("RESET_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{ $t("SAVE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api" :show_timeis_elapsed_time="true"
                :show_timeis_plaing_end_button="false" :highlight_targets="highlight_targets" :is_image_view="false"
                :kyou="kyou" :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_mi_plaing_end_button="true" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_attached_timeis="true" @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
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
        <NewBoardNameDialog v-if="kyou.typed_mi" :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @setted_new_board_name="(board_name: string) => update_board_name(board_name)"
            ref="new_board_name_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditMiViewProps } from './edit-mi-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue'
import moment from 'moment'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'

const new_board_name_dialog = ref<InstanceType<typeof NewBoardNameDialog> | null>(null);

const is_requested_submit = ref(false)

const props = defineProps<EditMiViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const show_kyou: Ref<boolean> = ref(false)
const mi_board_names: Ref<Array<string>> = ref(props.application_config.mi_default_board !== "" ? [props.application_config.mi_default_board] : [])

const mi_title: Ref<string> = ref(cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.title : "")
const mi_board_name: Ref<string> = ref(cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.board_name : "")
const mi_estimate_start_date: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("YYYY-MM-DD") : "")
const mi_estimate_start_time: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : "")
const mi_estimate_end_date: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("YYYY-MM-DD") : "")
const mi_estimate_end_time: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : "")
const mi_limit_date: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("YYYY-MM-DD") : "")
const mi_limit_time: Ref<string> = ref(cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : "")

watch(() => props.kyou, () => load())
load()

async function load(): Promise<void> {
    cloned_kyou.value = props.kyou.clone()
    await cloned_kyou.value.load_typed_datas()
    cloned_kyou.value.load_all()
    mi_title.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.title : ""
    mi_board_name.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.board_name : ""
    mi_estimate_start_date.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("YYYY-MM-DD") : ""
    mi_estimate_start_time.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : ""
    mi_estimate_end_date.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("YYYY-MM-DD") : ""
    mi_estimate_end_time.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : ""
    mi_limit_date.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("YYYY-MM-DD") : ""
    mi_limit_time.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : ""
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

function reset_estimate_start_date_time(): void {
    mi_estimate_start_date.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("YYYY-MM-DD") : ""
    mi_estimate_start_time.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : ""
}

function reset_estimate_end_date_time(): void {
    mi_estimate_end_date.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("YYYY-MM-DD") : ""
    mi_estimate_end_time.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : ""
}

function reset_limit_date_time(): void {
    mi_limit_date.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("YYYY-MM-DD") : ""
    mi_limit_time.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : ""
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
    mi_title.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.title : ""
    mi_board_name.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.board_name : ""
    mi_estimate_start_date.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("YYYY-MM-DD") : ""
    mi_estimate_start_time.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_start_time ? moment(cloned_kyou.value.typed_mi.estimate_start_time).format("HH:mm:ss") : ""
    mi_estimate_end_date.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("YYYY-MM-DD") : ""
    mi_estimate_end_time.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.estimate_end_time ? moment(cloned_kyou.value.typed_mi.estimate_end_time).format("HH:mm:ss") : ""
    mi_limit_date.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("YYYY-MM-DD") : ""
    mi_limit_time.value = cloned_kyou.value.typed_mi && cloned_kyou.value.typed_mi.limit_time ? moment(cloned_kyou.value.typed_mi.limit_time).format("HH:mm:ss") : ""
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
        if (mi.title === mi_title.value &&
            mi.board_name === mi_board_name.value &&
            (moment(mi.estimate_start_time).toDate().getTime() === moment(mi_estimate_start_date.value + " " + mi_estimate_start_time.value).toDate().getTime() || (mi.estimate_start_time == null && mi_estimate_start_date.value === "" && mi_estimate_start_time.value === "")) &&
            (moment(mi.estimate_end_time).toDate().getTime() === moment(mi_estimate_end_date.value + " " + mi_estimate_end_time.value).toDate().getTime() || (mi.estimate_end_time == null && mi_estimate_end_date.value === "" && mi_estimate_end_time.value === "")) &&
            (moment(mi.limit_time).toDate().getTime() === moment(mi_limit_date.value + " " + mi_limit_time.value).toDate().getTime() || (mi.limit_time == null && mi_limit_date.value === "" && mi_limit_time.value === ""))) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.mi_is_no_update
            error.error_message = i18n.global.t("MI_IS_NO_UPDATE_MESSAGE")
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

        // 更新後Mi情報を用意する
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
        const updated_mi = await mi.clone()
        updated_mi.title = mi_title.value
        updated_mi.board_name = mi_board_name.value
        updated_mi.estimate_start_time = estimate_start_time
        updated_mi.estimate_end_time = estimate_end_time
        updated_mi.limit_time = limit_time
        updated_mi.update_app = "gkill"
        updated_mi.update_device = gkill_info_res.device
        updated_mi.update_time = new Date(Date.now())
        updated_mi.update_user = gkill_info_res.user_id

        // 更新リクエストを飛ばす
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