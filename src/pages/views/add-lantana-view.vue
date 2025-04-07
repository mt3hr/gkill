<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ $t('ADD_LANTANA_TITLE') }}</span>
                </v-col>
                <v-spacer />
            </v-row>
        </v-card-title>
        <LantanaFlowersView :application_config="application_config" :gkill_api="gkill_api" :mood="mood"
            :editable="!is_requested_submit" ref="edit_lantana_flowers" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <label>{{ $t("LANTANA_DATE_TIME_TITLE") }}</label>
                <input class="input" type="date" v-model="related_date" :label="$t('LANTANA_DATE_TITLE')" :readonly="is_requested_submit" />
                <input class="input" type="time" v-model="related_time" :label="$t('LANTANA_TIME_TITLE')" :readonly="is_requested_submit" />
                <v-btn dark color="secondary" @click="reset_related_time()" :disabled="is_requested_submit">{{ $t('RESET_TITLE') }}</v-btn>
                <v-btn dark color="primary" @click="now_to_related_time()"
                    :disabled="is_requested_submit">{{ $t('CURRENT_DATE_TIME_TITLE') }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{ $t("RESET_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{ $t("SAVE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { ref, type Ref } from 'vue'

import type { KyouViewEmits } from './kyou-view-emits'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import moment from 'moment'
import LantanaFlowersView from './lantana-flowers-view.vue'
import { Lantana } from '@/classes/datas/lantana'
import type { AddLantanaViewProps } from './add-lantana-view-props'
import { AddLantanaRequest } from '@/classes/api/req_res/add-lantana-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const edit_lantana_flowers = ref<InstanceType<typeof LantanaFlowersView> | null>(null);

const is_requested_submit = ref(false)

const props = defineProps<AddLantanaViewProps>()
const emits = defineEmits<KyouViewEmits>()

const lantana: Ref<Lantana> = ref((() => {
    const lantana = new Lantana()
    lantana.related_time = new Date(Date.now())
    return lantana
})())
const mood: Ref<Number> = ref(lantana.value.mood)
const related_date: Ref<string> = ref(moment().format("YYYY-MM-DD"))
const related_time: Ref<string> = ref(moment().format("HH:mm:ss"))

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        if (!lantana.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_lantana_is_null
            error.error_message = t("CLIENT_LANTANA_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 日時必須入力チェック
        if (related_date.value === "" || related_time.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.lantana_related_time_is_blank
            error.error_message = t("LANTANA_DATE_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新がなかったらエラーメッセージを出力する
        if (lantana.value.mood === await edit_lantana_flowers.value?.get_mood()) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.lantana_is_no_update
            error.error_message = t("LANTANA_IS_NO_UPDATE_MESSAGE")
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

        // 追加するLantana情報を用意する
        const new_lantana = await lantana.value.clone()
        new_lantana.id = props.gkill_api.generate_uuid()
        new_lantana.mood = await edit_lantana_flowers.value!.get_mood()
        new_lantana.related_time = moment(related_date.value + " " + related_time.value).toDate()
        new_lantana.create_app = "gkill"
        new_lantana.create_device = gkill_info_res.device
        new_lantana.create_time = new Date(Date.now())
        new_lantana.create_user = gkill_info_res.user_id
        new_lantana.update_app = "gkill"
        new_lantana.update_device = gkill_info_res.device
        new_lantana.update_time = new Date(Date.now())
        new_lantana.update_user = gkill_info_res.user_id

        // 追加リクエストを飛ばす
        const req = new AddLantanaRequest()
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
        emits('requested_reload_list')
        emits('requested_close_dialog')
        return
    } finally {
        is_requested_submit.value = false
    }
}

function reset_related_time(): void {
    related_date.value = moment(lantana.value.related_time).format("YYYY-MM-DD")
    related_time.value = moment(lantana.value.related_time).format("HH:mm:ss")
}

function now_to_related_time(): void {
    related_date.value = moment().format("YYYY-MM-DD")
    related_time.value = moment().format("HH:mm:ss")
}

function reset(): void {
    mood.value = lantana.value.mood
    related_date.value = moment().format("YYYY-MM-DD")
    related_time.value = moment().format("HH:mm:ss")
}
</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>