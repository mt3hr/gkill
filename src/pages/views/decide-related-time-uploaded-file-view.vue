<template>
    <v-card>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <KyouListView class="decide_related_time_uploaded_files_kyou_list_view"
                    :show_timeis_plaing_end_button="false" :application_config="application_config"
                    :gkill_api="gkill_api" :matched_kyous="uploaded_kyous" :kyou_height="180" :width="400"
                    :list_height="app_content_height.valueOf() - 200" :query="new FindKyouQuery()"
                    :is_focused_list="false" :closable="false"
                    @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                    :last_added_tag="last_added_tag" :is_readonly_mi_check="true"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    :is_show_doc_image_toggle_button="false" :is_show_arrow_button="false" :show_rep_name="true"
                    :force_show_latest_kyou_info="true"
                    @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" :scroll_top="0"
                    :show_checkbox="false" :show_footer="false" :show_content_only="false"
                    @clicked_kyou="(...kyou: any[]) => focused_kyou = kyou[0] as Kyou"
                    @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                    @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                    @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                    @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0])"
                    @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                    @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                    @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                    @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                    @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                    @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                    @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                    @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                    @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                    ref="kyou_list_view" />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <div class="edit_idf_kyou_view_wrap">
                    <EditIDFKyouView v-if="focused_kyou" :application_config="application_config" :gkill_api="gkill_api"
                        :highlight_targets="[]" :kyou="focused_kyou" :last_added_tag="last_added_tag"
                        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                        @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                        @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0])"
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
                        @requested_reload_kyou="(...kyou: any[]) => {
                            reload_focused_kyou()
                            emits('requested_reload_kyou', kyou[0] as Kyou)
                        }" ref="edit_idf_kyou_view" />
                </div>
            </v-col>
        </v-row>
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { DecideRelatedTimeUploadedFileViewProps } from './decide-related-time-uploaded-file-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import EditIDFKyouView from './edit-idf-kyou-view.vue'
import KyouListView from './kyou-list-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import { computed, type Ref, ref } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

const enable_context_menu = ref(true)
const enable_dialog = ref(true)

defineProps<DecideRelatedTimeUploadedFileViewProps>()
const emits = defineEmits<KyouViewEmits>()
const focused_kyou: Ref<Kyou | null> = ref(null)

const kyou_height: Ref<number> = ref(180)
const kyou_height_px = computed(() => kyou_height.value ? kyou_height.value.toString().concat("px") : "0px")

async function reload_focused_kyou(): Promise<void> {
    if (!focused_kyou.value) {
        return
    }
    const updated_kyou = focused_kyou.value.clone()
    await updated_kyou.reload(false, true)
    await updated_kyou.load_all()
    focused_kyou.value = updated_kyou
}
</script>
<style lang="css">
.decide_related_time_uploaded_files_kyou_list_view .kyou_in_list .v-card {
    overflow: hidden;
}

.edit_idf_kyou_view_wrap {
    width: calc(400px + 8px);
    max-width: calc(400px + 8px);
    min-width: calc(400px + 8px);
}

.decide_related_time_uploaded_files_kyou_list_view .kyou .v-card {
    overflow: hidden !important;
    height: v-bind(kyou_height_px) !important;
    min-height: v-bind(kyou_height_px) !important;
    max-height: v-bind(kyou_height_px) !important;
    border-top: 1px solid silver;
}

.edit_idf_kyou_view_wrap .kyou {
    overflow: hidden !important;
    width: unset;
    max-width: unset;
    min-width: unset;
    border-top: 1px solid silver;
    height: unset !important;
    min-height: unset !important;
    max-height: unset !important;
}

.edit_idf_kyou_view_wrap .kyou_image {
    width: -webkit-fill-available !important;
    height: fit-content !important;
    max-width: -webkit-fill-available !important;
    max-height: 85vh !important;
}
</style>