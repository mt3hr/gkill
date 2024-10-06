<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>リポスト</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="rekyou()">リポスト</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
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
import type { ConfirmReKyouViewProps } from './confirm-re-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import KyouView from './kyou-view.vue'
import { ReKyou } from '@/classes/datas/re-kyou'
import { AddReKyouRequest } from '@/classes/api/req_res/add-re-kyou-request'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import router from '@/router'

const props = defineProps<ConfirmReKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const show_kyou: Ref<boolean> = ref(true)

async function rekyou(): Promise<void> {
    // セッションIDを取得する
    const session_id = window.localStorage.getItem("gkill_session_id")
    if (!session_id) {
        window.localStorage.removeItem("gkill_session_id")
        router.replace('/login')
        return
    }
    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    get_gkill_req.session_id = session_id
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

    // rekyou情報を用意する
    const new_rekyou = new ReKyou()
    new_rekyou.id = props.gkill_api.generate_uuid()
    new_rekyou.is_deleted = false
    new_rekyou.target_id = props.kyou.id
    new_rekyou.related_time = new Date(Date.now())
    new_rekyou.create_app = "gkill"
    new_rekyou.create_device = gkill_info_res.device
    new_rekyou.create_time = new Date(Date.now())
    new_rekyou.create_user = gkill_info_res.user_id
    new_rekyou.create_app = "gkill"
    new_rekyou.update_device = gkill_info_res.device
    new_rekyou.update_time = new Date(Date.now())
    new_rekyou.update_user = gkill_info_res.user_id

    // 追加リクエストを飛ばす
    const req = new AddReKyouRequest()
    req.session_id = session_id
    req.rekyou= new_rekyou
    const res = await props.gkill_api.add_rekyou(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    return
}
</script>
