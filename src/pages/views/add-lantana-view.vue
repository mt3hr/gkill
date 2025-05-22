<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t('ADD_LANTANA_TITLE') }}</span>
                </v-col>
                <v-spacer />
            </v-row>
        </v-card-title>
        <LantanaFlowersView :application_config="application_config" :gkill_api="gkill_api" :mood="mood"
            :editable="!is_requested_submit" ref="edit_lantana_flowers" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_related_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_date_string"
                                        :label="i18n.global.t('LANTANA_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="related_date_typed"
                                    @update:model-value="show_related_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_related_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_time_string"
                                        :label="i18n.global.t('LANTANA_TIME_TITLE')" min-width="120" readonly
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="related_time_string" format="24hr"
                                    @update:model-value="show_related_time_menu = false" />
                            </v-menu>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="pt-2">
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="reset_related_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_related_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>

        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{
                    i18n.global.t("RESET_TITLE")
                    }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{
                    i18n.global.t("SAVE_TITLE")
                    }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, ref, type Ref } from 'vue'

import type { KyouViewEmits } from './kyou-view-emits'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import moment from 'moment'
import LantanaFlowersView from './lantana-flowers-view.vue'
import { Lantana } from '@/classes/datas/lantana'
import type { AddLantanaViewProps } from './add-lantana-view-props'
import { AddLantanaRequest } from '@/classes/api/req_res/add-lantana-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/labs/components'
import delete_gkill_cache from '@/classes/delete-gkill-cache'

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
const related_date_typed: Ref<Date> = ref(moment().toDate())
const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
const related_time_string: Ref<string> = ref(moment().format("HH:mm:ss"))
const show_related_date_menu = ref(false)
const show_related_time_menu = ref(false)

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // データがちゃんとあるか確認。なければエラーメッセージを出力する
        if (!lantana.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.client_lantana_is_null
            error.error_message = i18n.global.t("CLIENT_LANTANA_IS_NULL_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 日時必須入力チェック
        if (related_date_string.value === "" || related_time_string.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.lantana_related_time_is_blank
            error.error_message = i18n.global.t("LANTANA_DATE_TIME_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新がなかったらエラーメッセージを出力する
        if (lantana.value.mood === await edit_lantana_flowers.value?.get_mood()) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.lantana_is_no_update
            error.error_message = i18n.global.t("LANTANA_IS_NO_UPDATE_MESSAGE")
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
        new_lantana.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
        new_lantana.create_app = "gkill"
        new_lantana.create_device = gkill_info_res.device
        new_lantana.create_time = new Date(Date.now())
        new_lantana.create_user = gkill_info_res.user_id
        new_lantana.update_app = "gkill"
        new_lantana.update_device = gkill_info_res.device
        new_lantana.update_time = new Date(Date.now())
        new_lantana.update_user = gkill_info_res.user_id

        // 追加リクエストを飛ばす
        await delete_gkill_cache(new_lantana.id)
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

function reset_related_date_time(): void {
    related_date_typed.value = moment(lantana.value.related_time).toDate()
    related_time_string.value = moment(lantana.value.related_time).format("HH:mm:ss")
}

function now_to_related_date_time(): void {
    related_date_typed.value = moment().toDate()
    related_time_string.value = moment().format("HH:mm:ss")
}

function reset(): void {
    mood.value = lantana.value.mood
    related_date_typed.value = moment().toDate()
    related_time_string.value = moment().format("HH:mm:ss")
}
</script>

<style lang="css" scoped></style>