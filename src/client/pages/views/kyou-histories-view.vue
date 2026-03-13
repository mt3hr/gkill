<template>
    <v-card>
        <div>
            <KyouView v-for="kyou in cloned_kyou.attached_histories" :application_config="application_config" :is_image_request_to_thumb_size="false"
                :key="kyou.update_time.getTime()" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :is_image_view="false" :kyou="kyou" :show_checkbox="false"
                :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_elapsed_time="false"
                :show_timeis_plaing_end_button="false" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_rep_name="true" :force_show_latest_kyou_info="false" :show_attached_timeis="false"
                :show_attached_tags="false" :show_attached_texts="false" :show_attached_notifications="false"
                v-on="crudRelayHandlers" :show_update_time="true"
                :show_related_time="false" />
        </div>
    </v-card>
</template>
<script lang="ts" setup>
import type { KyouHistoriesViewProps } from './kyou-histories-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { useKyouHistoriesView } from '@/classes/use-kyou-histories-view'

const props = defineProps<KyouHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    cloned_kyou,
    crudRelayHandlers,
} = useKyouHistoriesView({ props, emits })
</script>
