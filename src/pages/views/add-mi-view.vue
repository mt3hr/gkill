<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>Mi追加</span>
                </v-col>
            </v-row>
        </v-card-title>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>タイトル</label>
            </v-col>
            <v-col cols="auto">
                <input class="input text" type="text" v-model="mi_title" label="タイトル" autofocus />
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>板名</label>
            </v-col>
            <v-col cols="auto">
                <span>
                    <select class="select" v-model="mi_board_name">
                        <option v-for="board_name, index in mi_board_names">{{ board_name }}</option>
                    </select>
                </span>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="show_new_board_name_dialog()" icon="mdi-plus" dark size="small"></v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>開始日時</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="mi_estimate_start_date" label="開始日付" />
                <input class="input time" type="time" v-model="mi_estimate_start_time" label="開始時刻" />
            </v-col>
            <v-col cols="auto">
                <v-btn color="primary" @click="clear_estimate_start_date_time()">クリア</v-btn>
                <v-btn color="primary" @click="now_to_estimate_start_date_time()">現在日時</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>終了日時</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="mi_estimate_end_date" label="終了日付" />
                <input class="input time" type="time" v-model="mi_estimate_end_time" label="終了時刻" />
            </v-col>
            <v-col cols="auto">
                <v-btn color="primary" @click="clear_estimate_end_date_time()">クリア</v-btn>
                <v-btn color="primary" @click="now_to_estimate_end_date_time()">現在日時</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>期限日時</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="mi_limit_date" label="期限日付" />
                <input class="input time" type="time" v-model="mi_limit_time" label="期限時刻" />
            </v-col>
            <v-col cols="auto">
                <v-btn color="primary" @click="clear_limit_date_time()">クリア</v-btn>
                <v-btn color="primary" @click="now_to_limit_date_time()">現在日時</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="reset()">リセット</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="save()">保存</v-btn>
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
import type { AddMiViewProps } from './add-mi-view-props'
import { type Ref, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import { Mi } from '@/classes/datas/mi'
import KyouView from './kyou-view.vue'
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue'
import moment from 'moment'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import router from '@/router'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import { AddMiRequest } from '@/classes/api/req_res/add-mi-request'
import { GkillAPI } from '@/classes/api/gkill-api'

const new_board_name_dialog = ref<InstanceType<typeof NewBoardNameDialog> | null>(null);

const props = defineProps<AddMiViewProps>()
const emits = defineEmits<KyouViewEmits>()

const mi: Ref<Mi> = ref(new Mi())
const show_kyou: Ref<boolean> = ref(false)
const mi_board_names: Ref<Array<string>> = ref(new Array())

const mi_title: Ref<string> = ref(mi ? mi.value.title : "")
const mi_board_name: Ref<string> = ref(props.application_config.mi_default_board !== "" ? props.application_config.mi_default_board : "Inbox")
const mi_estimate_start_date: Ref<string> = ref(mi && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("YYYY-MM-DD") : "")
const mi_estimate_start_time: Ref<string> = ref(mi && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : "")
const mi_estimate_end_date: Ref<string> = ref(mi && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("YYYY-MM-DD") : "")
const mi_estimate_end_time: Ref<string> = ref(mi && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : "")
const mi_limit_date: Ref<string> = ref(mi && mi.value.limit_time ? moment(mi.value.limit_time).format("YYYY-MM-DD") : "")
const mi_limit_time: Ref<string> = ref(mi && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : "")

watch(() => props.application_config, () => load_mi_board_names)
load_mi_board_names()

async function load_mi_board_names(): Promise<void> {
    const req = new GetMiBoardRequest()
    req.session_id = props.gkill_api.get_session_id()

    const res = await props.gkill_api.get_mi_board_list(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        // emits('received_messages', res.messages)
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
    mi_estimate_start_date.value = mi && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("YYYY-MM-DD") : ""
    mi_estimate_start_time.value = mi && mi.value.estimate_start_time ? moment(mi.value.estimate_start_time).format("HH:mm:ss") : ""
    mi_estimate_end_date.value = mi && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("YYYY-MM-DD") : ""
    mi_estimate_end_time.value = mi && mi.value.estimate_end_time ? moment(mi.value.estimate_end_time).format("HH:mm:ss") : ""
    mi_limit_date.value = mi && mi.value.limit_time ? moment(mi.value.limit_time).format("YYYY-MM-DD") : ""
    mi_limit_time.value = mi && mi.value.limit_time ? moment(mi.value.limit_time).format("HH:mm:ss") : ""
}

async function save(): Promise<void> {
    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    if (!mi) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "クライアントのデータが変です"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 開始日時　片方だけ入力されていたらエラーチェック
    if (mi_estimate_start_date.value === "" || mi_estimate_start_time.value === "") {//どっちも入力されていなければOK。nullとして扱う
        if ((mi_estimate_start_date.value === "" && mi_estimate_start_time.value !== "") ||
            (mi_estimate_start_date.value !== "" && mi_estimate_start_time.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "開始日付付または開始時刻が入力されていません"
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }
    }

    // 終了日時　片方だけ入力されていたらエラーチェック
    if (mi_estimate_end_date.value === "" || mi_estimate_end_time.value === "") {//どっちも入力されていなければOK。nullとして扱う
        if ((mi_estimate_end_date.value === "" && mi_estimate_end_time.value !== "") ||
            (mi_estimate_end_date.value !== "" && mi_estimate_end_time.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "終了日付または終了時刻が入力されていません"
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }
    }

    // 期限日時　片方だけ入力されていたらエラーチェック
    if (mi_limit_date.value === "" || mi_limit_time.value === "") {//どっちも入力されていなければOK。nullとして扱う
        if ((mi_limit_date.value === "" && mi_limit_time.value !== "") ||
            (mi_limit_date.value !== "" && mi_limit_time.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "期限日付付または期限時刻が入力されていません"
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
        error.error_code = "//TODO"
        error.error_message = "Miが入力されていません"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    get_gkill_req.session_id = props.gkill_api.get_session_id()
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
    new_mi.id = props.gkill_api.generate_uuid()
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
    req.session_id = props.gkill_api.get_session_id()
    req.mi = new_mi
    const res = await props.gkill_api.add_mi(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits("registered_kyou", res.added_mi_kyou)
    emits('requested_reload_list')
    emits('requested_close_dialog')
    return
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
</style>