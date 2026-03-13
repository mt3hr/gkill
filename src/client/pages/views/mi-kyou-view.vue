<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height" :draggable="props.draggable"
        @dragstart="(...args: any[]) => on_drag_start(args[0] as DragEvent)">
        <v-row v-if="kyou.typed_mi" class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0" :style="mi_title_style">
                <table class="pa-0 ma-0">
                    <tr>
                        <td class="pa-0 ma-0">
                            <v-checkbox v-model="is_checked_mi" hide-details @click="clicked_mi_check()" />
                        </td>
                        <td class="pa-0 ma-0">
                            <div class="py-1 mi_title">{{ kyou.typed_mi.title }}</div>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-card-title>
                    <div class="py-1 mi_board_name">{{ kyou.typed_mi.board_name }}</div>
                </v-card-title>
            </v-col>
        </v-row>
        <div :style="{ 'padding-top': '30px' }">
            <div v-if="kyou.typed_mi && kyou.typed_mi.estimate_start_time">
                <span>{{ i18n.global.t("MI_START_DATE_TIME_TITLE") }}：</span>
                <span>{{ format_time(kyou.typed_mi.estimate_start_time) }}</span>
            </div>
            <div v-if="kyou.typed_mi && kyou.typed_mi.estimate_end_time">
                <span>{{ i18n.global.t("MI_END_DATE_TIME_TITLE") }}：</span>
                <span>{{ format_time(kyou.typed_mi.estimate_end_time) }}</span>
            </div>
            <div v-if="kyou.typed_mi && kyou.typed_mi.limit_time">
                <span>{{ i18n.global.t("MI_LIMIT_DATE_TIME_TITLE") }}：</span>
                <span>{{ format_time(kyou.typed_mi.limit_time) }}</span>
            </div>
        </div>
        <MiContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers"
            ref="context_menu" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import MiContextMenu from './mi-context-menu.vue'
import type { miKyouViewProps } from './mi-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { format_time } from '@/classes/format-date-time'
import { useMiKyouView } from '@/classes/use-mi-kyou-view'

const props = defineProps<miKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    cloned_kyou,
    is_checked_mi,
    mi_title_style,
    show_context_menu,
    clicked_mi_check,
    on_drag_start,
    crudRelayHandlers,
} = useMiKyouView({ props, emits })

defineExpose({ show_context_menu })
</script>
<style lang="css" scoped>
.mi_title_card {
    border: solid white 0px;
}
</style>