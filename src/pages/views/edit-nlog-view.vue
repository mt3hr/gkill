<template>
    <v-card v-if="cloned_kyou.typed_nlog" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>Nlog編集</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-if="kyou.typed_nlog" v-model="nlog_title_value" label="タイトル" autofocus />
        <v-text-field v-if="kyou.typed_nlog" v-model="nlog_shop_value" label="店名" />
        <v-text-field v-if="kyou.typed_nlog" v-model="nlog_amount_value" type="number" label="金額" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <label>日時</label>
                <input class="input date" type="date" v-model="related_date" label="日付" />
                <input class="input time" type="time" v-model="related_time" label="時刻" />
                <v-btn color="primary" @click="reset_related_date_time()">リセット</v-btn>
                <v-btn color="primary" @click="now_to_related_date_time()">現在日時</v-btn>
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
            <KyouView v-if="kyou.typed_nlog" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_plaing_end_button="true" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import moment from 'moment'
import { UpdateNlogRequest } from '@/classes/api/req_res/update-nlog-request'
import type { EditNlogViewProps } from './edit-nlog-view-props'
import type { Kyou } from '@/classes/datas/kyou'

const props = defineProps<EditNlogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const nlog_title_value: Ref<string> = ref(props.kyou.typed_nlog ? props.kyou.typed_nlog.title : "")
const nlog_amount_value: Ref<Number> = ref(props.kyou.typed_nlog ? props.kyou.typed_nlog.amount : 0)
const nlog_shop_value: Ref<string> = ref(props.kyou.typed_nlog ? props.kyou.typed_nlog.shop : "")

const related_date: Ref<string> = ref(moment(props.kyou.related_time).format("YYYY-MM-DD"))
const related_time: Ref<string> = ref(moment(props.kyou.related_time).format("HH:mm:ss"))
const show_kyou: Ref<boolean> = ref(false)

watch(() => props.kyou, () => load())
load()

async function load(): Promise<void> {
    cloned_kyou.value = props.kyou.clone()
    await cloned_kyou.value.load_all()
    nlog_title_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.title : ""
    nlog_amount_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.amount : 0
    nlog_shop_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.shop : ""
    related_date.value = moment(props.kyou.related_time).format("YYYY-MM-DD")
    related_time.value = moment(props.kyou.related_time).format("HH:mm:ss")
}

async function save(): Promise<void> {
    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    const nlog = props.kyou.typed_nlog
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
    get_gkill_req.session_id = props.gkill_api.get_session_id()
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

    // 更新後Kmemo情報を用意する
    const updated_nlog = await nlog.clone()
    updated_nlog.amount = nlog_amount_value.value
    updated_nlog.shop = nlog_shop_value.value
    updated_nlog.title = nlog_title_value.value
    updated_nlog.related_time = moment(related_date.value + " " + related_time.value).toDate()
    updated_nlog.update_app = "gkill"
    updated_nlog.update_device = gkill_info_res.device
    updated_nlog.update_time = new Date(Date.now())
    updated_nlog.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateNlogRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.nlog = updated_nlog
    const res = await props.gkill_api.update_nlog(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('updated_kyou', res.updated_nlog_kyou)
    emits('requested_reload_kyou', props.kyou)
    emits('requested_close_dialog')
    return
}

function now_to_related_date_time(): void {
    related_date.value = moment().format("YYYY-MM-DD")
    related_time.value = moment().format("HH:mm:ss")
}

function reset_related_date_time(): void {
    related_date.value = moment(props.kyou.related_time).format("YYYY-MM-DD")
    related_time.value = moment(props.kyou.related_time).format("HH:mm:ss")
}

function reset(): void {
    nlog_title_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.title : ""
    nlog_amount_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.amount : 0
    nlog_shop_value.value = props.kyou.typed_nlog ? props.kyou.typed_nlog.shop : ""
    related_date.value = moment(props.kyou.related_time).format("YYYY-MM-DD")
    related_time.value = moment(props.kyou.related_time).format("HH:mm:ss")

}
</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>