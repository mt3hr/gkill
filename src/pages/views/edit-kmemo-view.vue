<template>
    <v-card v-if="cloned_kyou.typed_kmemo" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>Kmemo編集</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-textarea v-model="kmemo_value" label="Kmemo" autofocus />
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
            <KyouView v-if="kyou.typed_kmemo" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_plaing_end_button="true" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_attached_timeis="true" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue'
import type { EditKmemoViewProps } from './edit-kmemo-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { Kyou } from '@/classes/datas/kyou'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateKmemoRequest } from '@/classes/api/req_res/update-kmemo-request'
import moment from 'moment'

const props = defineProps<EditKmemoViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const kmemo_value: Ref<string> = ref(cloned_kyou.value.typed_kmemo ? cloned_kyou.value.typed_kmemo.content : "")
const related_date: Ref<string> = ref(moment(cloned_kyou.value.related_time).format("YYYY-MM-DD"))
const related_time: Ref<string> = ref(moment(cloned_kyou.value.related_time).format("HH:mm:ss"))
const show_kyou: Ref<boolean> = ref(false)

watch(() => props.kyou, () => load())
load()

async function load(): Promise<void> {
    cloned_kyou.value = props.kyou.clone()
    await cloned_kyou.value.load_typed_datas()
    cloned_kyou.value.load_all()
    kmemo_value.value = cloned_kyou.value.typed_kmemo ? cloned_kyou.value.typed_kmemo.content : ""
    related_date.value = moment(cloned_kyou.value.related_time).format("YYYY-MM-DD")
    related_time.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
}

async function save(): Promise<void> {
    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    const kmemo = cloned_kyou.value.typed_kmemo
    if (!kmemo) {
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
        error.error_message = "日時が入力されていません"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 更新がなかったらエラーメッセージを出力する
    if (kmemo.content === kmemo_value.value) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "Kmemo更新されていません"
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

    // 更新後Kmemo情報を用意する
    const updated_kmemo = await kmemo.clone()
    updated_kmemo.content = kmemo_value.value
    updated_kmemo.related_time = moment(related_date.value + " " + related_time.value).toDate()
    updated_kmemo.update_app = "gkill"
    updated_kmemo.update_device = gkill_info_res.device
    updated_kmemo.update_time = new Date(Date.now())
    updated_kmemo.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateKmemoRequest()
    req.kmemo = updated_kmemo
    const res = await props.gkill_api.update_kmemo(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('updated_kyou', res.updated_kmemo_kyou)
    emits('requested_reload_kyou', props.kyou)
    emits('requested_close_dialog')
    return
}

function now_to_related_date_time(): void {
    related_date.value = moment().format("YYYY-MM-DD")
    related_time.value = moment().format("HH:mm:ss")
}

function reset_related_date_time(): void {
    related_date.value = moment(cloned_kyou.value.related_time).format("YYYY-MM-DD")
    related_time.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
}

function reset(): void {
    kmemo_value.value = cloned_kyou.value.typed_kmemo ? cloned_kyou.value.typed_kmemo.content : ""
    related_date.value = moment(cloned_kyou.value.related_time).format("YYYY-MM-DD")
    related_time.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
}
</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>