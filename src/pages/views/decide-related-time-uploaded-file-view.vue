<template>
    <v-card>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <KyouListView class="decide_related_time_uploaded_files_kyou_list_view"
                    :application_config="application_config" :gkill_api="gkill_api" :matched_kyous="uploaded_kyous"
                    :kyou_height="180" :width="400" :list_height="app_content_height.valueOf() - 200"
                    :query="new FindKyouQuery()" :is_focused_list="false" :closable="false"
                    @received_errors="(errors) => emits('received_errors', errors)" :last_added_tag="last_added_tag"
                    :is_readonly_mi_check="true" :enable_context_menu="enable_context_menu"
                    :enable_dialog="enable_dialog"
                    @received_messages="(messages) => emits('received_messages', messages)" :scroll_top="0 /* //TODO */"
                    :show_checkbox="false" :show_footer="false" :show_content_only="false"
                    @clicked_kyou="(kyou) => focused_kyou = kyou"
                    @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" ref="kyou_list_view" />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <div class="edit_idf_kyou_view_wrap">
                    <EditIDFKyouView v-if="focused_kyou" :application_config="application_config" :gkill_api="gkill_api"
                        :highlight_targets="[]" :kyou="focused_kyou" :last_added_tag="last_added_tag"
                        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                        ref="edit_idf_kyou_view" />
                </div>
            </v-col>
        </v-row>
    </v-card>
</template>
<script setup lang="ts">
import type { DecideRelatedTimeUploadedFileViewProps } from './decide-related-time-uploaded-file-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import EditIDFKyouView from './edit-idf-kyou-view.vue'
import KyouListView from './kyou-list-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import { computed, type Ref, ref } from 'vue'

const enable_context_menu = ref(true)
const enable_dialog = ref(true)

defineProps<DecideRelatedTimeUploadedFileViewProps>()
const emits = defineEmits<KyouViewEmits>()
const focused_kyou: Ref<Kyou | null> = ref(null)

const kyou_height: Ref<number> = ref(180)
const kyou_height_px = computed(() => kyou_height.value ? kyou_height.value.toString().concat("px") : "0px")
</script>
<style lang="css">
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
</style>