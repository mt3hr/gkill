<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>タグ編集</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" label="対象表示" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-model="tag_name" label="タグ" autofocus />
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
                :is_readonly_mi_check="true" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue'
import type { EditTagViewProps } from './edit-tag-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { UpdateTagRequest } from '@/classes/api/req_res/update-tag-request';
import router from '@/router';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { GkillError } from '@/classes/api/gkill-error';
import { GkillAPI } from '@/classes/api/gkill-api';
import type { Tag } from '@/classes/datas/tag';

const props = defineProps<EditTagViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_tag: Ref<Tag> = ref(props.tag.clone())
const tag_name: Ref<string> = ref(props.tag.tag)
const show_kyou: Ref<boolean> = ref(false)

watch(() => props.tag, () => load())
load()

async function load(): Promise<void> {
    cloned_tag.value = props.tag.clone()
    tag_name.value = cloned_tag.value.tag
}

async function save(): Promise<void> {
    // 更新がなかったらエラーメッセージを出力する
    if (cloned_tag.value.tag === tag_name.value) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "タグが更新されていません"
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

    // 更新後タグ情報を用意する
    const updated_tag = await cloned_tag.value.clone()
    updated_tag.tag = tag_name.value
    updated_tag.update_app = "gkill"
    updated_tag.update_device = gkill_info_res.device
    updated_tag.update_time = new Date(Date.now())
    updated_tag.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateTagRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.tag = updated_tag
    const res = await props.gkill_api.update_tag(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits("updated_tag", res.updated_tag)
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