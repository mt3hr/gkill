<template>
    <v-card elevation="0" @contextmenu.prevent.stop="show_context_menu" :width="width" :height="height"
        :draggable="effective_draggable" @dragstart="(e: DragEvent) => on_drag_start(e)">
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="is_checked_mi" hide-details @click="clicked_mi_check()" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-card-title>
                    <div class="py-1 mi_board_name">{{ mirekyou.board_name }}</div>
                </v-card-title>
            </v-col>
        </v-row>
        <KyouView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :is_image_request_to_thumb_size="false" :is_image_view="false"
            :kyou="target_kyou" :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true"
            :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
            :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true" :height="'unset'" :width="width"
            :is_readonly_mi_check="false" :enable_context_menu="false" :show_attached_timeis="false"
            :enable_dialog="enable_dialog" :show_rep_name="true" :force_show_latest_kyou_info="true"
            :show_update_time="false" :show_related_time="true" :show_attached_tags="true" :show_attached_texts="true"
            :show_attached_notifications="true"
            v-on="crudRelayHandlers"
            @dblclick.prevent.stop="() => { }" />
        <div>
            <div v-if="mirekyou.estimate_start_time">
                <span>{{ i18n.global.t("MI_START_DATE_TIME_TITLE") }}：</span>
                <span>{{ format_time(mirekyou.estimate_start_time) }}</span>
            </div>
            <div v-if="mirekyou.estimate_end_time">
                <span>{{ i18n.global.t("MI_END_DATE_TIME_TITLE") }}：</span>
                <span>{{ format_time(mirekyou.estimate_end_time) }}</span>
            </div>
            <div v-if="mirekyou.limit_time">
                <span>{{ i18n.global.t("MI_LIMIT_DATE_TIME_TITLE") }}：</span>
                <span>{{ format_time(mirekyou.limit_time) }}</span>
            </div>
        </div>
        <MiReKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers"
            ref="context_menu" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import MiReKyouContextMenu from './mi-re-kyou-context-menu.vue'
import KyouView from './kyou-view.vue'
import type { MiReKyouViewProps } from './mi-re-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { format_time } from '@/classes/format-date-time'
import { useMiReKyouView } from '@/classes/use-mi-re-kyou-view'

const props = defineProps<MiReKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    target_kyou,
    is_checked_mi,
    effective_draggable,
    show_context_menu,
    clicked_mi_check,
    on_drag_start,
    crudRelayHandlers,
} = useMiReKyouView({ props, emits })

defineExpose({ show_context_menu })
</script>
