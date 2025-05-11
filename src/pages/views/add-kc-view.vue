<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ $t("ADD_KC_TITLE") }}</span>
                </v-col>
                <v-spacer />
            </v-row>
        </v-card-title>
        <v-text-field v-model="title" :label="$t('KC_TITLE_TITLE')" autofocus :readonly="is_requested_submit" />
        <v-text-field type="number" v-model="num_value" :label="$t('KC_NUM_VALUE_TITLE')"
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <label>{{ $t("KC_DATE_TIME_TITLE") }}</label>
                <input class="input date" type="date" v-model="related_date" :label="$t('KC_DATE_TITLE')"
                    :readonly="is_requested_submit" />
                <input class="input time" type="time" v-model="related_time" :label="$t('KC_TIME_TITLE')"
                    :readonly="is_requested_submit" />
                <v-btn dark color="secondary" @click="reset_related_date_time()" :disabled="is_requested_submit">{{
                    $t("RESET_TITLE") }}</v-btn>
                <v-btn dark color="primary" @click="now_to_related_date_time()" :disabled="is_requested_submit">{{
                    $t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{ $t("RESET_TITLE")
                }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{ $t("SAVE_TITLE")
                }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditKCViewProps } from './edit-kc-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { KC } from '@/classes/datas/kc'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { AddKCRequest } from '@/classes/api/req_res/add-kc-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

import { i18n } from '@/i18n'

const is_requested_submit = ref(false)

const props = defineProps<EditKCViewProps>()
const emits = defineEmits<KyouViewEmits>()

const kc: Ref<KC> = ref(((): KC => {
    const kc = new KC()
    kc.related_time = moment().toDate()
    return kc
})())
const title: Ref<string> = ref(kc.value.title)
const num_value: Ref<number> = ref(kc.value.num_value)
const related_date: Ref<string> = ref(moment(kc.value.related_time).format("YYYY-MM-DD"))
const related_time: Ref<string> = ref(moment(kc.value.related_time).format("HH:mm:ss"))

function reset(): void {
    title.value = (kc.value.title)
    num_value.value = kc.value.num_value
    related_date.value = (moment(kc.value.related_time).format("YYYY-MM-DD"))
    related_time.value = (moment(kc.value.related_time).format("HH:mm:ss"))
}

function reset_related_date_time(): void {
    related_date.value = moment(kc.value.related_time).format("YYYY-MM-DD")
    related_time.value = moment(kc.value.related_time).format("HH:mm:ss")
}

function now_to_related_date_time(): void {
    related_date.value = moment().format("YYYY-MM-DD")
    related_time.value = moment().format("HH:mm:ss")
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        if (!kc.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_kc_is_null
            error.error_message = i18n.global.t("CLIENT_KC_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 日時必須入力チェック
        if (related_date.value === "" || related_time.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.kc_related_time_is_blank
            error.error_message = i18n.global.t("KC_RELATED_TIME_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // タイトル入力チェック
        if (title.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.kc_title_is_blank
            error.error_message = i18n.global.t("KC_TITLE_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 数値入力チェック
        if (num_value.value === null) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.kc_title_is_blank
            error.error_message = i18n.global.t("KC_NUM_VALUE_IS_BLANK_MESSAGE")
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

        // 更新後KC情報を用意する
        const new_kc = new KC()
        new_kc.id = props.gkill_api.generate_uuid()
        new_kc.title = title.value
        new_kc.num_value = num_value.value
        new_kc.related_time = moment(related_date.value + " " + related_time.value).toDate()
        new_kc.create_app = "gkill"
        new_kc.create_device = gkill_info_res.device
        new_kc.create_time = new Date(Date.now())
        new_kc.create_user = gkill_info_res.user_id
        new_kc.update_app = "gkill"
        new_kc.update_device = gkill_info_res.device
        new_kc.update_time = new Date(Date.now())
        new_kc.update_user = gkill_info_res.user_id

        // 追加リクエストを飛ばす
        const req = new AddKCRequest()
        req.kc = new_kc
        const res = await props.gkill_api.add_kc(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits("updated_kyou", res.added_kc_kyou)
        emits('requested_reload_list')
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}
</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>