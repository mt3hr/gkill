<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>TimeIs追加</span>
                </v-col>
                <v-spacer />
            </v-row>
        </v-card-title>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>タイトル</label>
            </v-col>
            <v-col cols="auto">
                <input class="input text" type="text" v-model="timeis_title" label="タイトル" autofocus />
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>開始日時</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="timeis_start_date" label="開始日付" />
                <input class="input time" type="time" v-model="timeis_start_time" label="開始時刻" />
            </v-col>
            <v-col cols="auto">
                <v-btn color="primary" @click="reset_start_date_time()">リセット</v-btn>
                <v-btn color="primary" @click="now_to_start_date_time()">現在日時</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>終了日時</label>
            </v-col>
            <v-col cols="auto">
                <input class="input date" type="date" v-model="timeis_end_date" label="終了日付" />
                <input class="input date" type="time" v-model="timeis_end_time" label="終了時刻" />
            </v-col>
            <v-col cols="auto">
                <v-btn color="primary" @click="reset_end_date_time()">リセット</v-btn>
                <v-btn color="primary" @click="clear_end_date_time()">クリア</v-btn>
                <v-btn color="primary" @click="now_to_end_date_time()">現在日時</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="reset()">リセット</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="() => save()">保存</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditTimeIsViewProps } from './edit-time-is-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { TimeIs } from '@/classes/datas/time-is'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { AddTimeisRequest } from '@/classes/api/req_res/add-timeis-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

const props = defineProps<EditTimeIsViewProps>()
const emits = defineEmits<KyouViewEmits>()

const timeis: Ref<TimeIs> = ref(((): TimeIs => {
    const timeis = new TimeIs()
    timeis.start_time = moment().toDate()
    timeis.end_time = null
    return timeis
})())
const timeis_title: Ref<string> = ref(timeis.value.title)
const timeis_start_date: Ref<string> = ref(moment(timeis.value.start_time).format("YYYY-MM-DD"))
const timeis_start_time: Ref<string> = ref(moment(timeis.value.start_time).format("HH:mm:ss"))
const timeis_end_date: Ref<string> = ref("")
const timeis_end_time: Ref<string> = ref("")

function reset(): void {
    timeis_title.value = (timeis.value.title)
    timeis_start_date.value = (moment(timeis.value.start_time).format("YYYY-MM-DD"))
    timeis_start_time.value = (moment(timeis.value.start_time).format("HH:mm:ss"))
    timeis_end_date.value = ("")
    timeis_end_time.value = ("")
}

function reset_start_date_time(): void {
    timeis_start_date.value = moment(timeis.value.start_time).format("YYYY-MM-DD")
    timeis_start_time.value = moment(timeis.value.start_time).format("HH:mm:ss")
}

function reset_end_date_time(): void {
    timeis_end_date.value = ""
    timeis_end_time.value = ""
}

function clear_end_date_time(): void {
    timeis_end_date.value = ""
    timeis_end_time.value = ""
}

function now_to_start_date_time(): void {
    timeis_start_date.value = moment().format("YYYY-MM-DD")
    timeis_start_time.value = moment().format("HH:mm:ss")
}

function now_to_end_date_time(): void {
    timeis_end_date.value = moment().format("YYYY-MM-DD")
    timeis_end_time.value = moment().format("HH:mm:ss")
}

async function save(): Promise<void> {
    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    if (!timeis.value) {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.client_timeis_is_null
        error.error_message = "クライアントのデータが変です"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 開始日時必須入力チェック
    if (timeis_start_date.value === "" || timeis_start_time.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.timeis_start_time_is_blank
        error.error_message = "開始日時が入力されていません"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 終了日時 片方だけ入力されていたらエラーチェック
    if (timeis_end_date.value === "" || timeis_end_time.value === "") {//どっちも入力されていなければOK。nullとして扱う
        if ((timeis_end_date.value === "" && timeis_end_time.value !== "") ||
            (timeis_end_date.value !== "" && timeis_end_time.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
            const error = new GkillError()
            error.error_code = GkillErrorCodes.timeis_end_time_is_blank
            error.error_message = "終了日付または終了時刻が入力されていません"
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
        error.error_message = "タイトルが入力されていません"
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

    // 更新後TimeIs情報を用意する
    let end_time: Date | null = null
    if (timeis_end_date.value !== "" && timeis_end_time.value !== "") {
        end_time = moment(timeis_end_date.value + " " + timeis_end_time.value).toDate()
    }
    const new_timeis = await timeis.value.clone()
    new_timeis.id = props.gkill_api.generate_uuid()
    new_timeis.title = timeis_title.value
    new_timeis.start_time = moment(timeis_start_date.value + " " + timeis_start_time.value).toDate()
    new_timeis.end_time = end_time
    new_timeis.create_app = "gkill"
    new_timeis.create_device = gkill_info_res.device
    new_timeis.create_time = new Date(Date.now())
    new_timeis.create_user = gkill_info_res.user_id
    new_timeis.update_app = "gkill"
    new_timeis.update_device = gkill_info_res.device
    new_timeis.update_time = new Date(Date.now())
    new_timeis.update_user = gkill_info_res.user_id

    // 追加リクエストを飛ばす
    const req = new AddTimeisRequest()
    req.timeis = new_timeis
    const res = await props.gkill_api.add_timeis(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits("updated_kyou", res.added_timeis_kyou)
    emits('requested_reload_list')
    emits('requested_close_dialog')
    return
}


</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>