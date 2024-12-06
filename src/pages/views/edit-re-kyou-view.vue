<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>Kyou編集</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
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
        <v-card v-if="show_kyou">
            <KyouView v-if="kyou.typed_rekyou" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[kyou.typed_rekyou.generate_info_identifer()]" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_plaing_end_button="true" :height="'100%'" :width="'100%'"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditReKyouViewProps } from './edit-re-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { ReKyou } from '@/classes/datas/re-kyou'
import { Kyou } from '@/classes/datas/kyou'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateKyouInfoRequest } from '@/classes/api/req_res/update-kyou-info-request'
import router from '@/router'
import moment from 'moment'
import { GkillAPI } from '@/classes/api/gkill-api'
import { UpdateReKyouRequest } from '@/classes/api/req_res/update-re-kyou-request'

const props = defineProps<EditReKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const related_date: Ref<string> = ref(moment(props.kyou.related_time).format("YYYY-MM-DD"))
const related_time: Ref<string> = ref(moment(props.kyou.related_time).format("HH:mm:ss"))
const show_kyou: Ref<boolean> = ref(true)

async function save(): Promise<void> {
    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    const rekyou = props.kyou.typed_rekyou
    if (!rekyou) {
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

    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    get_gkill_req.session_id = GkillAPI.get_instance().get_session_id()
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

    // 更新後ReKyou情報を用意する
    const updated_rekyou = rekyou.clone()
    updated_rekyou.related_time = moment(related_date.value + " " + related_time.value).toDate()
    updated_rekyou.update_app = "gkill"
    updated_rekyou.update_device = gkill_info_res.device
    updated_rekyou.update_time = new Date(Date.now())
    updated_rekyou.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateReKyouRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.rekyou = updated_rekyou
    const res = await props.gkill_api.update_rekyou(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('updated_kyou', res.updated_rekyou_kyou)
    emits('requested_close_dialog')
    return
}
</script>
