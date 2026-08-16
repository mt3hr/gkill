<template>
    <div @dblclick="show_kyou_dialog()" @click.prevent="onRootClick()"
        :key="kyou.id" class="kyou_view_root" :class="'kyou_'.concat(kyou.id)">
        <!-- 保存後の引き直し中。中身は消さずに重ねる。
             消すと一覧の行がちらつき、高さ可変の詳細ビューでは高さが跳ねる。
             体裁はPluginKyouの読み込み中表示に合わせる -->
        <div v-if="show_reloading_indicator" class="kyou_reloading">
            <v-progress-circular indeterminate size="16" />
        </div>
        <div v-if="!show_content_only" :class="kyou_class">
            <!-- 一覧の行で1行ellipsisに詰めるための容れ物。折り返しの抑止はコンテナに掛ける必要があるので、
                 個々のタグではなくここを kyou-list-view.vue が :deep() で掴む -->
            <div class="kyou_attached_tags">
                <AttachedTag v-for="attached_tag in cloned_kyou.attached_tags" :tag="attached_tag" :key="attached_tag.id"
                    :application_config="application_config" :gkill_api="gkill_api" :kyou="cloned_kyou"
                    :highlight_targets="highlight_targets"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers" />
            </div>
            <div v-if="show_attached_timeis">
                <AttachedTimeIsPlaing v-for="attached_timeis_plaing in cloned_kyou.attached_timeis_kyou"
                    :key="attached_timeis_plaing.id" :timeis_kyou="attached_timeis_plaing"
                    :application_config="application_config" :gkill_api="gkill_api" :kyou="cloned_kyou"
                    :highlight_targets="highlight_targets"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers" />
            </div>
            <v-row class="pa-0 ma-0" @contextmenu.prevent="async (e: PointerEvent) => show_context_menu(e)"
                :class="kyou_class">
                <v-col v-if="show_related_time" class="kyou_related_time pa-0 ma-0" cols="auto">
                    {{ related_time }}
                </v-col>
                <v-col v-if="show_update_time" class="kyou_update_time pa-0 ma-0" cols="auto">
                    {{ update_time }}
                </v-col>
                <v-spacer />
                <v-col v-if="show_rep_name" class="kyou_rep_name pa-0 ma-0" cols="auto">
                    <span>{{ rep_name }}</span>
                </v-col>
            </v-row>
        </div>
        <div :class="kyou_class">
            <!-- 種別データを取りに行っている間の表示。PluginKyouの読み込み中表示に体裁を合わせる。
                 読み終わるまで下の各ビューを出さないのは、インジケータと中身が同時に出て
                 高さが二重になり行からはみ出すのを防ぐため -->
            <div v-if="show_loading_indicator" class="kyou_loading">
                <v-progress-circular indeterminate size="20" />
            </div>
            <template v-else>
                <KmemoView v-if="cloned_kyou.typed_kmemo" :kmemo="cloned_kyou.typed_kmemo" :draggable=draggable
                    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                    :kyou="cloned_kyou" :height="height" :width="width" :max-width="width"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="kmemo_view" />
                <KCView v-if="cloned_kyou.typed_kc" :kc="cloned_kyou.typed_kc" :application_config="application_config"
                    :draggable=draggable :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="cloned_kyou"
                    :height="height" :width="width"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="kc_view" />
                <miKyouView v-if="cloned_kyou.typed_mi" :mi="cloned_kyou.typed_mi" :application_config="application_config"
                    :draggable=draggable :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="cloned_kyou"
                    :height="height" :width="width" :is_readonly_mi_check="is_readonly_mi_check"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="mi_view" />
                <NlogView v-if="cloned_kyou.typed_nlog" :nlog="cloned_kyou.typed_nlog" :draggable=draggable
                    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                    :kyou="cloned_kyou" :height="height" :width="width"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="nlog_view" />
                <LantanaView v-if="cloned_kyou.typed_lantana" :lantana="cloned_kyou.typed_lantana" :draggable=draggable
                    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                    :kyou="cloned_kyou" :height="height" :width="width"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="lantana_view" />
                <TimeIsView v-if="cloned_kyou.typed_timeis" :timeis="cloned_kyou.typed_timeis" :draggable=draggable
                    :show_timeis_elapsed_time="show_timeis_elapsed_time"
                    :show_timeis_plaing_end_button="show_timeis_plaing_end_button" :application_config="application_config"
                    :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="cloned_kyou"
                    :height="height" :width="width"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="timeis_view" />
                <URLogView v-if="cloned_kyou.typed_urlog" :urlog="cloned_kyou.typed_urlog" :draggable=draggable
                    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                    :kyou="cloned_kyou" :height="height" :width="width"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="urlog_view" />
                <IDFKyouView v-if="cloned_kyou.typed_idf_kyou" :idf_kyou="cloned_kyou.typed_idf_kyou" :draggable=draggable
                    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                    :kyou="cloned_kyou" :height="height" :width="width"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :enable_md_link_dialog="enable_md_link_dialog" :is_image_request_to_thumb_size="is_image_request_to_thumb_size"
                    v-on="crudRelayHandlers"
                    ref="idf_kyou_view" />
                <ReKyouView v-if="cloned_kyou.typed_rekyou" :rekyou="cloned_kyou.typed_rekyou" :draggable=draggable
                    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                    :kyou="cloned_kyou" :height="height" :width="width"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="rekyou_view" />
                <MiReKyouView v-if="cloned_kyou.typed_mirekyou" :mirekyou="cloned_kyou.typed_mirekyou" :draggable=draggable
                    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                    :kyou="cloned_kyou" :height="height" :width="width" :is_readonly_mi_check="is_readonly_mi_check"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="mirekyou_view" />
                <GitCommitLogView v-if="cloned_kyou.typed_git_commit_log" :git_commit_log="cloned_kyou.typed_git_commit_log"
                    :draggable=draggable :application_config="application_config" :gkill_api="gkill_api"
                    :highlight_targets="highlight_targets" :kyou="cloned_kyou"
                    :height="height" :width="width" :enable_context_menu="enable_context_menu"
                    :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="git_commit_log_view" />
                <PluginHtmlView v-if="cloned_kyou.typed_plugin"
                    :kyou="cloned_kyou" :application_config="application_config"
                    :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                    :height="height" :width="width"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    v-on="crudRelayHandlers"
                    ref="plugin_html_view" />
            </template>
        </div>
        <div v-if="!show_content_only">
            <AttachedText v-for="attached_text in cloned_kyou.attached_texts" :text="attached_text"
                :key="attached_text.id" :application_config="application_config" :gkill_api="gkill_api"
                :kyou="cloned_kyou" :highlight_targets="highlight_targets"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="crudRelayHandlers" />
        </div>
        <div v-if="!show_content_only">
            <AttachedNotification v-for="attached_notification in cloned_kyou.attached_notifications"
                :key="attached_notification.id" :notification="attached_notification"
                :application_config="application_config" :gkill_api="gkill_api" :kyou="cloned_kyou"
                :highlight_targets="highlight_targets"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="crudRelayHandlers" />
        </div>    </div>
</template>
<script setup lang="ts">
import AttachedTag from './attached-tag.vue'
import AttachedText from './attached-text.vue'
import AttachedTimeIsPlaing from './attached-time-is-plaing.vue'
import AttachedNotification from './attached-notification.vue'
import GitCommitLogView from './git-commit-log-view.vue'
import IDFKyouView from './idf-kyou-view.vue'
import KmemoView from './kmemo-view.vue'
import KCView from './kc-view.vue'
import LantanaView from './lantana-view.vue'
import miKyouView from './mi-kyou-view.vue'
import NlogView from './nlog-view.vue'
import ReKyouView from './re-kyou-view.vue'
import MiReKyouView from './mi-re-kyou-view.vue'
import TimeIsView from './time-is-view.vue'
import URLogView from './ur-log-view.vue'
import PluginHtmlView from './plugin-html-view.vue'

import type { KyouViewEmits } from './kyou-view-emits'
import type { KyouViewProps } from './kyou-view-props'
import { useKyouView } from '@/classes/use-kyou-view'

const props = defineProps<KyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // Template refs
    kmemo_view,
    kc_view,
    mi_view,
    nlog_view,
    lantana_view,
    timeis_view,
    urlog_view,
    idf_kyou_view,
    rekyou_view,
    mirekyou_view,
    git_commit_log_view,
    plugin_html_view,

    // State
    cloned_kyou,

    // Computed
    related_time,
    update_time,
    rep_name,
    kyou_class,
    show_loading_indicator,
    show_reloading_indicator,

    // Business logic
    show_context_menu,
    show_kyou_dialog,
    onRootClick,

    // Event relay objects
    crudRelayHandlers,
} = useKyouView({ props, emits })
</script>
<style lang="css" scoped>
/* コンテナ自身にも塗らないと、インライン要素であるタグ/TimeIsチップの
   まわりや行間の余白から地の色(ダークテーマだと#212121)が透けて黒帯になる */
.highlighted_kyou,
.highlighted_kyou>* {
    background-color: rgb(var(--v-theme-highlight));
}

.kyou_related_time,
.kyou_update_time,
.kyou_rep_name,
.kyou_device,
.kyou_data_type {
    font-size: small;
    color: silver;
}

/* 日時が出るまでの間も1行ぶんの高さを確保しておく。
   後から生えると参照先(ReKyou/MiReKyou)の中身が押し下げられて跳ねる */
.kyou_related_time,
.kyou_update_time {
    min-height: 1.5em;
}

/* 読み込み中表示はPluginKyou(.plugin-loading)と同じ体裁にする。
   v-progress-circularは色をcurrentColorから取るのでスピナーもグレーになる */
.kyou_loading {
    padding: 8px;
    font-size: 0.85em;
    color: gray;
}

/* 引き直し中のスピナーを重ねるための基準。
   直下の子はどれもstatic配置なので、ここにrelativeを足しても配置は変わらない */
.kyou_view_root {
    position: relative;
}

/* 引き直し中は中身を残したまま右上に重ねる。
   行(.kyou_in_list)は高さ固定でoverflow:hiddenなので、行内に収まる位置に置く。
   クリックとコンテキストメニューを奪わないようpointer-eventsは切る */
.kyou_reloading {
    position: absolute;
    top: 2px;
    right: 2px;
    z-index: 1;
    color: gray;
    pointer-events: none;
}
</style>
