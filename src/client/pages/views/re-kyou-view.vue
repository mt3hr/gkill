<template>
    <v-card elevation="0" @contextmenu.prevent.stop="show_context_menu" :width="width" :height="height">
        <!--
            内側のコンテキストメニューは殺す。
            生かすと参照先Kyou（Kmemo等）のメニューが先に出てしまい、
            ReKyou自身の編集・削除にどの経路からも到達できなくなる。MiReKyouと同じ扱い。
        -->
        <!-- 参照先が消えているときの終端表示。出さないと読み込み中表示のまま止まってしまう -->
        <div v-if="is_target_not_found" class="rekyou_not_found">
            {{ i18n.global.t('NOT_FOUND_REKYOU_TARGET_ERROR_MESSAGE') }}
        </div>
        <KyouView v-else :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets" :is_image_request_to_thumb_size="false"
            :is_image_view="false" :kyou="target_kyou" :show_checkbox="false"
            :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
            :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_elapsed_time="true"
            :show_timeis_plaing_end_button="true" :height="height" :width="width" :is_readonly_mi_check="false"
            :enable_context_menu="false" :show_attached_timeis="false" :enable_dialog="enable_dialog"
            :show_rep_name="true" :force_show_latest_kyou_info="true" :show_update_time="false"
            :show_related_time="true" :show_attached_tags="true" :show_attached_texts="true"
            :show_attached_notifications="true"
            v-on="crudRelayHandlers"
            @dblclick.prevent.stop="() => { }" />
        <ReKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" ref="context_menu"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
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
    is_target_not_found,
    show_context_menu,
    crudRelayHandlers,
} = useReKyouView({ props, emits })

defineExpose({ show_context_menu })
</script>
<style lang="css" scoped>
/* 参照先なしの表示はPluginKyou(.plugin-error)と同じ体裁にする */
.rekyou_not_found {
    padding: 8px;
    font-size: 0.85em;
    color: gray;
}
</style>
