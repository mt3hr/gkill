<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>TimeIs編集</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>タイトル</label>
            </v-col>
            <v-col cols="auto">
                <input class="input" type="text" v-model="timeis_title" label="タイトル" />
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>開始日時</label>
            </v-col>
            <v-col cols="auto">
                <input class="input" type="date" v-model="timeis_start_date" label="開始日付" />
                <input class="input" type="time" v-model="timeis_start_time" label="開始時刻" />
            </v-col>
            <v-col cols="auto">
                <v-btn color="primary" @click="clear_start_date_time()">クリア</v-btn>
                <v-btn color="primary" @click="now_to_start_date_time()">現在日時</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>終了日時</label>
            </v-col>
            <v-col cols="auto">
                <input class="input" type="date" v-model="timeis_end_date" label="終了日付" />
                <input class="input" type="time" v-model="timeis_end_time" label="終了時刻" />
            </v-col>
            <v-col cols="auto">
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
                <v-btn color="primary" @click="save()">保存</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[kyou.generate_info_identifer()]" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { computed, type Ref, ref } from 'vue'
import type { EditTimeIsViewProps } from './edit-time-is-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import { TimeIs } from '@/classes/datas/time-is'
import KyouView from './kyou-view.vue'
import moment from 'moment'
import { useDisplay } from 'vuetify'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateTimeisRequest } from '@/classes/api/req_res/update-timeis-request'
import router from '@/router'
import { GkillAPI } from '@/classes/api/gkill-api'

const props = defineProps<EditTimeIsViewProps>()
const emits = defineEmits<KyouViewEmits>()

const timeis_title: Ref<string> = ref(props.kyou.typed_timeis!.title)
const timeis_start_date: Ref<string> = ref(moment(props.kyou.typed_timeis!.start_time).format("YYYY-MM-DD"))
const timeis_start_time: Ref<string> = ref(moment(props.kyou.typed_timeis!.start_time).format("HH:mm:ss"))
const timeis_end_date: Ref<string> = ref(moment(props.kyou.typed_timeis!.end_time).format("YYYY-MM-DD"))
const timeis_end_time: Ref<string> = ref(moment(props.kyou.typed_timeis!.end_time).format("HH:mm:ss"))

const show_kyou: Ref<boolean> = ref(false)

function reset(): void {
    timeis_title.value = props.kyou.typed_timeis!.title
    timeis_start_date.value = moment(props.kyou.typed_timeis!.start_time).format("YYYY-MM-DD")
    timeis_start_time.value = moment(props.kyou.typed_timeis!.start_time).format("HH:mm:ss")
    timeis_end_date.value = moment(props.kyou.typed_timeis!.end_time).format("YYYY-MM-DD")
    timeis_end_time.value = moment(props.kyou.typed_timeis!.end_time).format("HH:mm:ss")
}

function clear_start_date_time(): void {
    timeis_start_date.value = ""
    timeis_start_time.value = ""
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
    const timeis = props.kyou.typed_timeis
    if (!timeis) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "クライアントのデータが変です"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 開始日時必須入力チェック
    if (timeis_start_date.value === "" || timeis_start_time.value === "") {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "開始日時が入力されていません"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }
    // 終了日時　片方だけ入力されていたらエラーチェック
    if (timeis_end_date.value === "" || timeis_end_time.value === "") {//どっちも入力されていなければOK。nullとして扱う
        if ((timeis_end_date.value === "" && timeis_end_time.value !== "") ||
            (timeis_end_date.value !== "" && timeis_end_time.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "終了日付または終了時刻が入力されていません"
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }
    }

    // 更新がなかったらエラーメッセージを出力する
    if (timeis.title === timeis_title.value &&
        moment(timeis.start_time) === (moment(timeis_start_date.value + timeis_start_time.value)) &&
        moment(timeis.end_time) === moment(timeis_end_date.value + timeis_end_time.value)) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "TimeIsが更新されていません"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    get_gkill_req.session_id = GkillAPI.get_instance().get_session_id()
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

    // 更新後TimeIs情報を用意する
    let end_time: Date | null = null
    if (timeis_end_date.value !== "" && timeis_end_time.value !== "") {
        end_time = moment(timeis_end_date.value + timeis_end_time.value).toDate()
    }
    const updated_timeis = await timeis.clone()
    updated_timeis.title = timeis_title.value
    updated_timeis.start_time = moment(timeis_start_date.value + timeis_start_time.value).toDate()
    updated_timeis.end_time = end_time
    updated_timeis.update_app = "gkill"
    updated_timeis.update_device = gkill_info_res.device
    updated_timeis.update_time = new Date(Date.now())
    updated_timeis.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateTimeisRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.timeis = updated_timeis
    const res = await props.gkill_api.update_timeis(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits("updated_kyou", res.updated_timeis_kyou)
    return
}


</script>
<style lang="css" scoped>
.input {
    border: solid 1px silver;
}
</style>