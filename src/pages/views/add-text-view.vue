<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>テキスト追加</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-textarea v-model="text_value" label="テキスト" />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="() => save()">保存</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="false" :show_attached_timeis="true"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { nextTick, type Ref, ref } from 'vue'
import type { AddTextViewProps } from './add-text-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { Text } from '@/classes/datas/text'
import KyouView from './kyou-view.vue'
import { AddTextRequest } from '@/classes/api/req_res/add-text-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'

const props = defineProps<AddTextViewProps>()
const emits = defineEmits<KyouViewEmits>()

const show_kyou: Ref<boolean> = ref(false)
const text_value: Ref<string> = ref("")


async function save(): Promise<void> {
    // 値がなかったらエラーメッセージを出力する
    if (text_value.value === "") {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "テキストが未入力です"
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

    // テキスト情報を用意する
    const new_text = new Text()
    new_text.text = text_value.value
    new_text.id = props.gkill_api.generate_uuid()
    new_text.is_deleted = false
    new_text.target_id = props.kyou.id
    new_text.related_time = new Date(Date.now())
    new_text.create_app = "gkill"
    new_text.create_device = gkill_info_res.device
    new_text.create_time = new Date(Date.now())
    new_text.create_user = gkill_info_res.user_id
    new_text.update_app = "gkill"
    new_text.update_app = "gkill"
    new_text.update_device = gkill_info_res.device
    new_text.update_time = new Date(Date.now())
    new_text.update_user = gkill_info_res.user_id

    // 追加リクエストを飛ばす
    const req = new AddTextRequest()
    req.text = new_text
    const res = await props.gkill_api.add_text(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_reload_kyou', props.kyou)
    emits('requested_close_dialog')
    return
}
</script>
