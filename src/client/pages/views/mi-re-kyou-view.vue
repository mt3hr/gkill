<template>
    <!--
        一覧の行では高さを指定しない。
        行の高さは外側KyouViewのヘッダ(related_time / rep_name)と分け合うことになるので、
        行高をそのまま受け取ると必ずヘッダのぶんだけはみ出して日時が切れる。
        内容なりの高さにしておけば、ヘッダが出ても出なくても行に収まる。
    -->
    <v-card class="mirekyou_card" elevation="0" @contextmenu.prevent.stop="show_context_menu" :width="width"
        :height="is_compact ? undefined : height" :draggable="effective_draggable"
        @dragstart="(e: DragEvent) => onDragStart(e)">
        <div class="mirekyou_head">
            <!-- 既定のdensityだと56pxあり、板名と日時と合わせると行に収まらない -->
            <v-checkbox class="mirekyou_check" v-model="is_checked_mi" hide-details density="compact"
                @click="clicked_mi_check()" :readonly="is_requested_submit" />
            <!-- 既存の記録をタスクにしたものであることをMiと区別できるようにする -->
            <v-icon class="mirekyou_mark" size="16">mdi-subdirectory-arrow-right</v-icon>
            <div class="py-1 mi_title mirekyou_summary" :title="target_summary">{{ target_summary }}</div>
            <v-card-title class="mirekyou_board">
                <div class="py-1 mi_board_name">{{ mirekyou.board_name }}</div>
            </v-card-title>
        </div>
        <div class="mirekyou_times">
            <!-- 一覧の行には1行しか入らないので、先に来る日時だけを出す -->
            <div v-if="is_compact && primary_time">
                <span>{{ i18n.global.t(primary_time.label_key) }}：</span>
                <span>{{ format_time(primary_time.time) }}</span>
            </div>
            <template v-if="!is_compact">
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
            </template>
        </div>
        <div v-if="!is_compact" class="mirekyou_target">
            <!-- 参照先が消えているときの終端表示。出さないと読み込み中表示のまま止まってしまう -->
            <div v-if="is_target_not_found" class="mirekyou_not_found">
                {{ i18n.global.t('NOT_FOUND_MI_REKYOU_TARGET_ERROR_MESSAGE') }}
            </div>
            <KyouView v-else :application_config="application_config" :gkill_api="gkill_api"
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
    is_requested_submit,
    is_checked_mi,
    target_summary,
    is_target_not_found,
    effective_draggable,
    is_compact,
    primary_time,
    show_context_menu,
    clicked_mi_check,
    onDragStart,
    crudRelayHandlers,
} = useMiReKyouView({ props, emits })

defineExpose({ show_context_menu })
</script>
<style lang="css" scoped>
/*
 * 一覧の行は高さが固定でoverflow:hiddenなので、はみ出した分は切り落とされる。
 * headと日時の高さを先に確定させ、余りを参照先に配ることで日時が切れないようにする。
 */
.mirekyou_card {
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.mirekyou_head {
    flex: 0 0 auto;
    display: flex;
    align-items: center;
    gap: 2px;
}

/* 参照先の本文は1行に収め、溢れたら三点リーダにする。全文はtitle属性で読める */
.mirekyou_summary {
    flex: 1 1 auto;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

/*
 * v-checkboxのルートは.v-input--horizontalのgridで、中身のトラックがminmax(0,1fr)。
 * min-content幅が0になるのでflexの縮小がそのまま効き、参照先の要約が長いほど
 * 取り分を持っていかれて幅0まで潰れ、チェックボックスが要約の下に隠れる。
 * 他の固定要素と同じく縮ませない
 */
.mirekyou_check {
    flex: 0 0 auto;
}

.mirekyou_mark {
    flex: 0 0 auto;
}

/* 字面はMi行に合わせてv-card-titleのままにし、行に収めるためpaddingだけ落とす */
.mirekyou_board {
    flex: 0 0 auto;
    padding: 0;
}

/* margin-top:autoで下端に寄せる。related_timeが出ても出なくても切れない */
.mirekyou_times {
    flex: 0 0 auto;
    margin-top: auto;
}

.mirekyou_target {
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
}

/* 参照先なしの表示はPluginKyou(.plugin-error)と同じ体裁にする */
.mirekyou_not_found {
    padding: 8px;
    font-size: 0.85em;
    color: gray;
}
</style>
