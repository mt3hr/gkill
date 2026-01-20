<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("EDIT_TAG_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-model="tag_name" :label="i18n.global.t('TAG_TITLE')" autofocus
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{
                    i18n.global.t("SAVE_TITLE")
                }}</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api" :is_image_request_to_thumb_size="false"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou"
                :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="true" :show_attached_timeis="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_update_time="false"
                :show_related_time="true" :show_attached_tags="true" :show_attached_texts="true"
                :show_attached_notifications="true"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref, watch } from 'vue'
import type { EditTagViewProps } from './edit-tag-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { UpdateTagRequest } from '@/classes/api/req_res/update-tag-request';
import { GkillError } from '@/classes/api/gkill-error';
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'

const is_requested_submit = ref(false)

const props = defineProps<EditTagViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_tag: Ref<Tag> = ref(props.tag.clone())
const tag_name: Ref<string> = ref(props.tag.tag)
const show_kyou: Ref<boolean> = ref(false)

watch(() => props.tag, () => load())
load()

async function load(): Promise<void> {
    cloned_tag.value = props.tag.clone()
    cloned_tag.value.attached_histories[0]
    tag_name.value = cloned_tag.value.tag
}

async function save(): Promise<void> {
    try {
        is_requested_submit.value = true
        // 値がなかったらエラーメッセージを出力する
        if (tag_name.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.tag_is_blank
            error.error_message = i18n.global.t("TAG_IS_BLANK_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新がなかったらエラーメッセージを出力する
        if (cloned_tag.value.tag === tag_name.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.tag_is_no_update
            error.error_message = i18n.global.t("TAG_IS_NO_UPDATE_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return
        }

        // 更新後タグ情報を用意する
        const updated_tag = await cloned_tag.value.clone()
        updated_tag.tag = tag_name.value
        updated_tag.update_app = "gkill"
        updated_tag.update_device = props.application_config.device
        updated_tag.update_time = new Date(Date.now())
        updated_tag.update_user = props.application_config.user_id

        // 更新リクエストを飛ばす
        await delete_gkill_kyou_cache(updated_tag.id)
        await delete_gkill_kyou_cache(updated_tag.target_id)
        const req = new UpdateTagRequest()
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
    } finally {
        is_requested_submit.value = false
    }
}
</script>

<style lang="css" scoped></style>