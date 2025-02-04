<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>テキスト編集</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-textarea v-model="text_value" label="テキスト" autofocus />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="save()">保存</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
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
import type { EditTextViewProps } from './edit-text-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { UpdateTextRequest } from '@/classes/api/req_res/update-text-request';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { GkillError } from '@/classes/api/gkill-error';
import type { Text } from '@/classes/datas/text';

const props = defineProps<EditTextViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_text: Ref<Text> = ref(props.text.clone())
const text_value: Ref<string> = ref(cloned_text.value.text)
const show_kyou: Ref<boolean> = ref(false)

watch(() => props.text, () => load())
load()

async function load(): Promise<void> {
    cloned_text.value = props.text.clone()
    text_value.value = cloned_text.value.text
}

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

    // 更新がなかったらエラーメッセージを出力する
    if (cloned_text.value.text === text_value.value) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "テキストが更新されていません"
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

    // 更新後テキスト情報を用意する
    const updated_text = await cloned_text.value.clone()
    updated_text.text = text_value.value
    updated_text.update_app = "gkill"
    updated_text.update_device = gkill_info_res.device
    updated_text.update_time = new Date(Date.now())
    updated_text.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateTextRequest()
    req.text = updated_text
    const res = await props.gkill_api.update_text(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits("updated_text", res.updated_text)
    emits('requested_reload_kyou', props.kyou)
    emits('requested_close_dialog')
    return
}
</script>

<style lang="css" scoped>
.input.date,
.input.time,
.input.text {
    border: solid 1px silver;
}
</style>