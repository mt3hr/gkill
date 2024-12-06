<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>Nlog追加</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-if="nlog" v-model="nlog_title_value" label="タイトル" />
        <v-text-field v-if="nlog" v-model="nlog_shop_value" label="店名" />
        <v-text-field v-if="nlog" v-model="nlog_amount_value" type="number" label="金額" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <label>日時</label>
                <input class="input" type="date" v-model="related_date" label="日付" />
                <input class="input" type="time" v-model="related_time" label="時刻" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="save()">保存</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import type { AddNlogViewProps } from './add-nlog-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { type Ref, ref } from 'vue'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import router from '@/router'
import moment from 'moment'
import { UpdateNlogRequest } from '@/classes/api/req_res/update-nlog-request'
import { Nlog } from '@/classes/datas/nlog'
import { AddNlogRequest } from '@/classes/api/req_res/add-nlog-request'
import { GkillAPI } from '@/classes/api/gkill-api'

const props = defineProps<AddNlogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const nlog: Nlog = new Nlog()
const nlog_title_value: Ref<string> = ref(nlog ? nlog.title : "")
const nlog_amount_value: Ref<Number> = ref(nlog ? nlog.amount : 0)
const nlog_shop_value: Ref<string> = ref(nlog ? nlog.shop : "")

const related_date: Ref<string> = ref(moment().format("YYYY-MM-DD"))
const related_time: Ref<string> = ref(moment().format("HH:mm:ss"))
const show_kyou: Ref<boolean> = ref(false)

async function save(): Promise<void> {
    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    if (!nlog) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "クライアントのデータが変です"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 日時必須入力チェック
    if (related_date.value === "" || related_time.value === "") {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "開始日時が入力されていません"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 更新がなかったらエラーメッセージを出力する
    if (nlog_amount_value.value === nlog.amount &&
        nlog_shop_value.value === nlog.shop &&
        nlog_title_value.value === nlog.title) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "Nlog更新されていません"
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

    // 更新後Kmemo情報を用意する
    const new_nlog = await nlog.clone()
    new_nlog.amount = nlog_amount_value.value
    new_nlog.shop = nlog_shop_value.value
    new_nlog.title = nlog_title_value.value
    new_nlog.related_time = moment(related_date.value + " " + related_time.value).toDate()
    new_nlog.create_app = "gkill"
    new_nlog.create_device = gkill_info_res.device
    new_nlog.create_time = new Date(Date.now())
    new_nlog.create_user = gkill_info_res.user_id
    new_nlog.update_app = "gkill"
    new_nlog.update_device = gkill_info_res.device
    new_nlog.update_time = new Date(Date.now())
    new_nlog.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new AddNlogRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.nlog = new_nlog
    const res = await props.gkill_api.add_nlog(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('registered_kyou', res.added_nlog_kyou)
    emits('requested_close_dialog')
    return
}
</script>