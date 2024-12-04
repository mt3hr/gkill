<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>URLog編集</span>
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
                <input class="input" type="text" v-model="urlog_title" label="タイトル" />
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>URL</label>
            </v-col>
            <v-col cols="auto">
                <input class="input" type="text" v-model="urlog_url" label="URL" />
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto">
                <label>日時</label>
            </v-col>
            <v-col cols="auto">
                <input class="input" type="date" v-model="urlog_related_date" label="日付" />
                <input class="input" type="time" v-model="urlog_related_time" label="時刻" />
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
                :show_timeis_plaing_end_button="true" :highlight_targets="[kyou.generate_info_identifer()]"
                :is_image_view="false" :kyou="kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
                :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_urlog_plaing_end_button="true"
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
import type { EditURLogViewProps } from './edit-ur-log-view-props'
import { URLog } from '@/classes/datas/ur-log'
import moment from 'moment'
import { useDisplay } from 'vuetify'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateURLogRequest } from '@/classes/api/req_res/update-ur-log-request'
import router from '@/router'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { GkillAPI } from '@/classes/api/gkill-api'

const props = defineProps<EditURLogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const urlog_title: Ref<string> = ref(props.kyou.typed_urlog!.title)
const urlog_url: Ref<string> = ref(props.kyou.typed_urlog!.url)
const urlog_related_date: Ref<string> = ref(moment(props.kyou.typed_urlog!.related_time).format("YYYY-MM-DD"))
const urlog_related_time: Ref<string> = ref(moment(props.kyou.typed_urlog!.related_time).format("HH:mm:ss"))

const show_kyou: Ref<boolean> = ref(false)

function reset(): void {
    urlog_title.value = props.kyou.typed_urlog!.title
    urlog_url.value = props.kyou.typed_urlog!.url
    urlog_related_date.value = moment(props.kyou.typed_urlog!.related_time).format("YYYY-MM-DD")
    urlog_related_time.value = moment(props.kyou.typed_urlog!.related_time).format("HH:mm:ss")
}

async function save(): Promise<void> {
    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    const urlog = props.kyou.typed_urlog
    if (!urlog) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "クライアントのデータが変です"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 日時必須入力チェック
    if (urlog_related_date.value === "" || urlog_related_time.value === "") {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "日時が入力されていません"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 更新がなかったらエラーメッセージを出力する
    if (urlog.title === urlog_title.value &&
        moment(urlog.related_time) === (moment(urlog_related_date.value + urlog_related_time.value)) &&
        moment(urlog.related_time) === moment(urlog_related_date.value + urlog_related_time.value)) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "URLogが更新されていません"
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

    // 更新後URLog情報を用意する
    const updated_urlog = await urlog.clone()
    updated_urlog.title = urlog_title.value
    updated_urlog.url = urlog_url.value
    updated_urlog.related_time = moment(urlog_related_date.value + urlog_related_time.value).toDate()
    updated_urlog.update_time = new Date(Date.now())
    updated_urlog.update_app = "gkill"
    updated_urlog.update_device = gkill_info_res.device
    updated_urlog.update_time = new Date(Date.now())
    updated_urlog.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateURLogRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.urlog = updated_urlog
    const res = await props.gkill_api.update_urlog(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits("updated_kyou", res.updated_urlog_kyou)
    return
}
</script>
<style lang="css" scoped>
.input {
    border: solid 1px silver;
}
</style>
