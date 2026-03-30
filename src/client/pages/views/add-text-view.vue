<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_TEXT_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-textarea v-model="text_value" :label="i18n.global.t('TEXT_TITLE')" autofocus
            :rules="[(v: string) => !!v || i18n.global.t('REQUIRED_FIELD_MESSAGE')]"
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
                :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="false" :show_attached_timeis="true"
                :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_update_time="false"
                :show_related_time="true" @deleted_kyou="(deleted_kyou: Kyou) => emits('deleted_kyou', deleted_kyou)"
                @deleted_tag="(deleted_tag: Tag) => emits('deleted_tag', deleted_tag)"
                @deleted_text="(deleted_text: Text) => emits('deleted_text', deleted_text)"
                @deleted_notification="(deleted_notification: Notification) => emits('deleted_notification', deleted_notification)"
                @registered_kyou="(registered_kyou: Kyou) => emits('registered_kyou', registered_kyou)"
                @registered_tag="(registered_tag: Tag) => emits('registered_tag', registered_tag)"
                @registered_text="(registered_text: Text) => emits('registered_text', registered_text)"
                @registered_notification="(registered_notification: Notification) => emits('registered_notification', registered_notification)"
                @updated_kyou="(updated_kyou: Kyou) => emits('updated_kyou', updated_kyou)"
                @updated_tag="(updated_tag: Tag) => emits('updated_tag', updated_tag)"
                @updated_text="(updated_text: Text) => emits('updated_text', updated_text)"
                @updated_notification="(updated_notification: Notification) => emits('updated_notification', updated_notification)"
                @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous: Kyou[], checked: boolean) => emits('requested_update_check_kyous', kyous, checked)" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AddTextViewProps } from './add-text-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import { useAddTextView } from '@/classes/use-add-text-view'

const props = defineProps<AddTextViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // State
    is_requested_submit,
    show_kyou,
    text_value,

    // Methods
    save,
} = useAddTextView({ props, emits })
</script>
