<template>
    <v-card @contextmenu.prevent.stop="() => {/*show_context_menu*/ }" :width="width" :height="height">
        <KyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets" :is_image_request_to_thumb_size="false"
            :is_image_view="false" :kyou="target_kyou" :show_checkbox="false"
            :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
            :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_elapsed_time="true"
            :show_timeis_plaing_end_button="true" :height="height" :width="width" :is_readonly_mi_check="false"
            :enable_context_menu="enable_context_menu" :show_attached_timeis="false" :enable_dialog="enable_dialog"
            :show_rep_name="true" :force_show_latest_kyou_info="true" :show_update_time="false"
            :show_related_time="true" :show_attached_tags="true" :show_attached_texts="true"
            :show_attached_notifications="true"
            @deleted_kyou="crudRelayHandlers['deleted_kyou']"
            @deleted_tag="crudRelayHandlers['deleted_tag']"
            @deleted_text="crudRelayHandlers['deleted_text']"
            @deleted_notification="crudRelayHandlers['deleted_notification']"
            @registered_kyou="crudRelayHandlers['registered_kyou']"
            @registered_tag="crudRelayHandlers['registered_tag']"
            @registered_text="crudRelayHandlers['registered_text']"
            @registered_notification="crudRelayHandlers['registered_notification']"
            @updated_kyou="crudRelayHandlers['updated_kyou']"
            @updated_tag="crudRelayHandlers['updated_tag']"
            @updated_text="crudRelayHandlers['updated_text']"
            @updated_notification="crudRelayHandlers['updated_notification']"
            @received_errors="crudRelayHandlers['received_errors']"
            @received_messages="crudRelayHandlers['received_messages']"
            @requested_reload_kyou="crudRelayHandlers['requested_reload_kyou']"
            @requested_reload_list="crudRelayHandlers['requested_reload_list']"
            @requested_update_check_kyous="crudRelayHandlers['requested_update_check_kyous']"
            @requested_open_rykv_dialog="crudRelayHandlers['requested_open_rykv_dialog']"
            @dblclick.prevent.stop="() => { }" />
        <ReKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" ref="context_menu"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @deleted_kyou="crudRelayHandlers['deleted_kyou']"
            @deleted_tag="crudRelayHandlers['deleted_tag']"
            @deleted_text="crudRelayHandlers['deleted_text']"
            @deleted_notification="crudRelayHandlers['deleted_notification']"
            @registered_kyou="crudRelayHandlers['registered_kyou']"
            @registered_tag="crudRelayHandlers['registered_tag']"
            @registered_text="crudRelayHandlers['registered_text']"
            @registered_notification="crudRelayHandlers['registered_notification']"
            @updated_kyou="crudRelayHandlers['updated_kyou']"
            @updated_tag="crudRelayHandlers['updated_tag']"
            @updated_text="crudRelayHandlers['updated_text']"
            @updated_notification="crudRelayHandlers['updated_notification']"
            @received_errors="crudRelayHandlers['received_errors']"
            @received_messages="crudRelayHandlers['received_messages']"
            @requested_reload_kyou="crudRelayHandlers['requested_reload_kyou']"
            @requested_reload_list="crudRelayHandlers['requested_reload_list']"
            @requested_update_check_kyous="crudRelayHandlers['requested_update_check_kyous']"
            @requested_open_rykv_dialog="crudRelayHandlers['requested_open_rykv_dialog']" />
    </v-card>
</template>
<script setup lang="ts">
import type { ReKyouViewProps } from './re-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import ReKyouContextMenu from './re-kyou-context-menu.vue'
import KyouView from './kyou-view.vue'
import { useReKyouView } from '@/classes/use-re-kyou-view'

const props = defineProps<ReKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    target_kyou,
    show_context_menu,
    crudRelayHandlers,
} = useReKyouView({ props, emits })

defineExpose({ show_context_menu })
</script>