<template>
    <v-card class="pa-2" variant="flat">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_TEXT_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :readonly="is_requested_submit"
                        :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-textarea v-model="text_value" :label="i18n.global.t('TEXT_TITLE')" autofocus
            :rules="[(v: string) => !!v || i18n.global.t('REQUIRED_FIELD_MESSAGE')]"
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0 gkill-dialog-actions">
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
                :show_related_time="true" v-on="crudRelayHandlers" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AddTextViewProps } from './add-text-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { useAddTextView } from '@/classes/use-add-text-view'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

const props = defineProps<AddTextViewProps>()
const emits = defineEmits<KyouViewEmits>()

const crudRelayHandlers = build_kyou_view_relay(emits)

const {
    // State
    is_requested_submit,
    show_kyou,
    text_value,

    // Methods
    save,
} = useAddTextView({ props, emits })
</script>
