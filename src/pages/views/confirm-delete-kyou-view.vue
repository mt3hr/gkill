<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>Kyou削除</span>
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
                <v-btn color="primary" @click="delete_kyou()">削除</v-btn>
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
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import type { ConfirmDeleteKyouViewProps } from './confirm-delete-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { type Ref, ref } from 'vue'
import KyouView from './kyou-view.vue'
import router from '@/router';
import { UpdateKyouInfoRequest } from '@/classes/api/req_res/update-kyou-info-request';
import { GkillAPI } from '@/classes/api/gkill-api';

const props = defineProps<ConfirmDeleteKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const show_kyou: Ref<boolean> = ref(true)

async function delete_kyou(): Promise<void> {
    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    get_gkill_req.session_id = GkillAPI.get_instance().get_session_id()
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

    // 更新後Kyou情報を用意する
    const updated_kyou = await props.kyou.clone()
    updated_kyou.is_deleted = true
    updated_kyou.update_app = "gkill"
    updated_kyou.update_device = gkill_info_res.device
    updated_kyou.update_time = new Date(Date.now())
    updated_kyou.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateKyouInfoRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.kyou = updated_kyou
    const res = await props.gkill_api.update_kyou_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('deleted_kyou', res.updated_kyou)
    return
}
</script>
