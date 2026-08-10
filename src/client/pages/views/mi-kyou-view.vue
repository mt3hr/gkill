<template>
    <v-card elevation="0" @contextmenu.prevent="show_context_menu" :width="width" :height="height" :draggable="effective_draggable"
        @dragstart="(e: DragEvent) => onDragStart(e)">
        <div v-if="kyou.typed_mi" class="mi_head">
            <v-checkbox class="mi_check" v-model="is_checked_mi" hide-details @click="clicked_mi_check()"
                :readonly="is_requested_submit" />
            <div class="py-1 mi_title" :title="kyou.typed_mi.title">{{ kyou.typed_mi.title }}</div>
            <v-card-title class="mi_board">
                <div class="py-1 mi_board_name">{{ kyou.typed_mi.board_name }}</div>
            </v-card-title>
        </div>
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
import type { MiKyouViewProps } from './mi-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { format_time } from '@/classes/format-date-time'
import { useMiKyouView } from '@/classes/use-mi-kyou-view'

const props = defineProps<MiKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    cloned_kyou: _cloned_kyou,
    is_requested_submit,
    is_checked_mi,
    effective_draggable,
    show_context_menu,
    clicked_mi_check,
    onDragStart,
    crudRelayHandlers,
} = useMiKyouView({ props, emits })

defineExpose({ show_context_menu })
</script>
<style lang="css" scoped>
/*
 * チェックボックス・タイトル・板名を1行に収める。
 * v-checkboxのルートは.v-input--horizontalのgridで中身のトラックがminmax(0,1fr)なので、
 * min-content幅が0になりflexの縮小がそのまま効く。タイトルが長いほど縮小の取り分を
 * 持っていかれ、チェックボックスが幅0まで潰れてタイトルの下に隠れてしまう。
 * 縮んでよいのはタイトルだけにして、溢れたら三点リーダにする(全文はtitle属性で読める)。
 * MiReKyou(.mirekyou_head)と同じ作りに揃えてある
 */
.mi_head {
    display: flex;
    align-items: center;
}

.mi_check {
    flex: 0 0 auto;
}

.mi_title {
    flex: 1 1 auto;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.mi_board {
    flex: 0 0 auto;
}
</style>