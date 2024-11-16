<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>Lantana編集</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <LantanaFlowersView :application_config="application_config" :gkill_api="gkill_api" :editable="true"
            :mood="mood" ref="edit_lantana_flowers" />
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
import { LantanaTextData } from '@/classes/lantana/lantana-text-data'
import { ref, type Ref } from 'vue'

import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateLantanaRequest } from '@/classes/api/req_res/update-lantana-request'
import router from '@/router'
import moment from 'moment'
import LantanaFlowersView from './lantana-flowers-view.vue'
import { Lantana } from '@/classes/datas/lantana'
import type { AddLantanaViewProps } from './add-lantana-view-props'
import { AddLantanaRequest } from '@/classes/api/req_res/add-lantana-request'
import { GkillAPI } from '@/classes/api/gkill-api'

const edit_lantana_flowers = ref<InstanceType<typeof LantanaFlowersView> | null>(null);

const props = defineProps<AddLantanaViewProps>()
const emits = defineEmits<KyouViewEmits>()

const lantana: Lantana = new Lantana()
const mood: Ref<Number> = ref(lantana!.mood)
const related_date: Ref<string> = ref(moment().format("YYYY-MM-DD"))
const related_time: Ref<string> = ref(moment().format("HH:mm:ss"))
const show_kyou: Ref<boolean> = ref(false)

async function save(): Promise<void> {
    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    if (!lantana) {
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
    if (lantana.mood === await edit_lantana_flowers.value?.get_mood()) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "Lantanaが更新されていません"
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

    // 追加するLantana情報を用意する
    const new_lantana = await lantana.clone()
    new_lantana.mood = await edit_lantana_flowers.value!.get_mood()
    new_lantana.related_time = moment(related_date.value + related_time.value).toDate()
    new_lantana.create_app = "gkill"
    new_lantana.create_device = gkill_info_res.device
    new_lantana.create_time = new Date(Date.now())
    new_lantana.create_app = "gkill"
    new_lantana.update_device = gkill_info_res.device
    new_lantana.update_time = new Date(Date.now())
    new_lantana.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new AddLantanaRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.lantana = new_lantana
    const res = await props.gkill_api.add_lantana(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('registered_kyou', res.added_lantana_kyou)
    return
}
</script>
<style lang="css" scoped>
.input {
    border: solid 1px silver;
}
</style>