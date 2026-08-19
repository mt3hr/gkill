<template>
    <v-card class="dnote_view" :class="{ fill_height_mode: fill_height }" variant="flat">
        <v-overlay v-model="is_loading" :content-class="'dnote_progress_overlay'" class="align-center justify-center"
            contained persistent>
            <v-progress-circular indeterminate color="primary" class="align-center justify-center" />
            <div v-if="is_fetching_from_api || getted_kyous_count !== target_kyous_count"
                class="align-center justify-center">
                <div class="align-center justify-center overlay_message">
                    {{ i18n.global.t('DNOTE_GETTING_DATA') }}
                </div>
                <div v-if="!is_fetching_from_api" class="align-center justify-center overlay_message">
                    {{ getted_kyous_count }}/{{ target_kyous_count }}
                </div>
            </div>
            <div v-if="!is_fetching_from_api && getted_kyous_count === target_kyous_count"
                class="align-center justify-center">
                <div class="align-center justify-center overlay_message">
                    {{ i18n.global.t('DNOTE_CALCURATING') }}
                </div>
                <div class="align-center justify-center overlay_message">
                    {{ finished_aggregate_task }}/{{ estimate_aggregate_task }}
                </div>
                <div class="align-center justify-center overlay_message">{{ i18n.global.t('DNOTE_PLEASE_WAIT_MESSAGE')
                    }}</div>
            </div>
        </v-overlay>
        <v-tabs v-if="dnote_definitions.length > 1 || editable" v-model="current_definition_index" show-arrows
            :center-active="false">
            <v-tab v-for="(def, i) in dnote_definitions" :key="i" :value="i">
                {{ def.name }}
            </v-tab>
            <v-tooltip :text="i18n.global.t('ADD_DNOTE_DEFINITION_TITLE')">
                <template v-slot:activator="{ props }">
                    <v-btn v-if="editable" v-bind="props" icon="mdi-plus" size="small" variant="text"
                        class="align-self-center ml-1" @click="add_definition" />
                </template>
            </v-tooltip>
        </v-tabs>
        <v-row v-if="editable && dnote_definitions.length > 0" class="pa-2 ma-0" align="center"
            style="max-height: fit-content">
            <v-col class="pa-0 ma-0">
                <v-text-field v-model="dnote_definitions[current_definition_index].name"
                    :label="i18n.global.t('DNOTE_DEFINITION_NAME_LABEL')" density="compact" hide-details />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <v-tooltip :text="i18n.global.t('DELETE_DNOTE_DEFINITION_TITLE')">
                    <template v-slot:activator="{ props }">
                        <v-btn v-if="dnote_definitions.length > 1" v-bind="props" icon="mdi-delete" size="small"
                            variant="text" @click="delete_current_definition" />
                    </template>
                </v-tooltip>
            </v-col>
        </v-row>
        <h1>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto pa-0 ma-0">
                    <span>{{ start_date_str }}</span>
                    <span v-if="end_date_str !== '' && start_date_str != end_date_str">～</span>
                    <span v-if="end_date_str !== '' && start_date_str != end_date_str">{{ end_date_str }}</span>
                    <span v-if="start_date_str === '' && !(end_date_str !== '' && start_date_str != end_date_str)">{{
                        i18n.global.t("DNOTE_WHOLE_PERIOD_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto pa-0 ma-0" v-if="!editable">
                    <v-tooltip :text="i18n.global.t('TOOLTIP_DOWNLOAD')">
                        <template v-slot:activator="{ props }">
                            <v-btn v-bind="props" :disabled="!loaded_kyous" icon="mdi-download-circle-outline"
                                @click="download_kyous_json" />
                        </template>
                    </v-tooltip>
                </v-col>
            </v-row>
        </h1>
        <v-window v-model="current_definition_index" :touch="false">
            <v-window-item v-for="(def, i) in dnote_definitions" :key="i" :value="i" :eager="true">
                <div class="dnote-scroll-wrap">
                    <DnoteItemTableView :application_config="application_config" :gkill_api="gkill_api"
                        :editable="editable" v-model="dnote_definitions[i].items"
                        v-on="crudRelayHandlers"
                        @finish_a_aggregate_task="increment_finished_aggregate_task"
                        :ref="(el) => set_item_table_ref(i, el)" />
                    <DnoteTrendGraphTableView :application_config="application_config" :gkill_api="gkill_api"
                        :editable="editable" v-if="dnote_definitions[i].trends" v-model="dnote_definitions[i].trends"
                        v-on="errorMessageRelayHandlers" @finish_a_aggregate_task="increment_finished_aggregate_task"
                        :ref="(el) => set_trend_table_ref(i, el)" />
                    <DnoteCorrelationGraphTableView :application_config="application_config" :gkill_api="gkill_api"
                        :editable="editable" v-if="dnote_definitions[i].correlations"
                        v-model="dnote_definitions[i].correlations"
                        v-on="errorMessageRelayHandlers"
                        @finish_a_aggregate_task="increment_finished_aggregate_task"
                        :ref="(el) => set_correlation_table_ref(i, el)" />
                    <DnoteListTableView :application_config="application_config" :gkill_api="gkill_api"
                        :editable="editable" v-if="dnote_definitions[i].lists" v-model="dnote_definitions[i].lists"
                        v-on="crudRelayHandlers"
                        @finish_a_aggregate_task="increment_finished_aggregate_task"
                        :ref="(el) => set_list_table_ref(i, el)" />
                </div>
            </v-window-item>
        </v-window>
        <v-avatar v-if="editable" :style="floating_action_button_style()" color="primary" class="position-fixed-dnote">
            <v-menu transition="slide-x-transition">
                <template v-slot:activator="{ props }">
                    <v-btn color="white" icon="mdi-plus" variant="text" v-bind="props" />
                </template>
                <v-list>
                    <v-list-item @click="add_dnote_item_dialog?.show()">
                        <v-list-item-title>{{ i18n.global.t("ADD_DNOTE_ITEM_MENU_TITLE") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="add_dnote_list_dialog?.show()">
                        <v-list-item-title>{{ i18n.global.t("ADD_DNOTE_LIST_MENU_TITLE") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="add_dnote_trend_graph_dialog?.show()">
                        <v-list-item-title>{{ i18n.global.t("ADD_DNOTE_TREND_GRAPH_MENU_TITLE") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="add_dnote_correlation_graph_dialog?.show()">
                        <v-list-item-title>{{ i18n.global.t("ADD_DNOTE_CORRELATION_GRAPH_MENU_TITLE") }}</v-list-item-title>
                    </v-list-item>
                </v-list>
            </v-menu>
        </v-avatar>
        <v-card-action v-if="editable" class="dnote_actions">
            <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ i18n.global.t("APPLY_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{
                        i18n.global.t("CANCEL_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddDnoteListDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_add_dnote_list_query="(query: DnoteListQuery) => onRequestedAddDnoteListQuery(query)"
            ref="add_dnote_list_dialog" />
        <AddDnoteItemDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_add_dnote_item="(item: DnoteItemData) => onRequestedAddDnoteItem(item)"
            ref="add_dnote_item_dialog" />
        <AddDnoteTrendGraphDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_add_dnote_trend_graph="(query: DnoteTrendGraphQuery) => onRequestedAddDnoteTrendGraph(query)"
            ref="add_dnote_trend_graph_dialog" />
        <AddDnoteCorrelationGraphDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_add_dnote_correlation_graph="(query: DnoteCorrelationGraphQuery) => onRequestedAddDnoteCorrelationGraph(query)"
            ref="add_dnote_correlation_graph_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type DnoteViewProps } from '@/pages/views/dnote-view-props'
import DnoteItemTableView from './dnote-item-table-view.vue'
import DnoteListTableView from './dnote-list-table-view.vue'
import DnoteTrendGraphTableView from './dnote-trend-graph-table-view.vue'
import DnoteCorrelationGraphTableView from './dnote-correlation-graph-table-view.vue'
import AddDnoteListDialog from '../../pages/dialogs/add-dnote-list-dialog.vue'
import AddDnoteItemDialog from '../../pages/dialogs/add-dnote-item-dialog.vue'
import AddDnoteTrendGraphDialog from '../../pages/dialogs/add-dnote-trend-graph-dialog.vue'
import AddDnoteCorrelationGraphDialog from '../../pages/dialogs/add-dnote-correlation-graph-dialog.vue'
import { type DnoteEmits } from '@/pages/views/dnote-emits'
import { useDnoteView } from '@/classes/use-dnote-view'
import type DnoteListQuery from "@/pages/views/dnote-list-query"
import type DnoteTrendGraphQuery from "@/pages/views/dnote-trend-graph-query"
import type { DnoteCorrelationGraphQuery } from "@/classes/dnote/dnote-correlation"
import type DnoteItem from "@/classes/dnote/dnote-item"
type DnoteItemData = DnoteItem

const props = defineProps<DnoteViewProps>()
const emits = defineEmits<DnoteEmits>()

const {
    // Template refs
    add_dnote_list_dialog,
    add_dnote_item_dialog,
    add_dnote_trend_graph_dialog,
    add_dnote_correlation_graph_dialog,

    // View ref helpers
    set_item_table_ref,
    set_list_table_ref,
    set_trend_table_ref,
    set_correlation_table_ref,

    // State
    dnote_definitions,
    current_definition_index,
    is_loading,
    is_fetching_from_api,
    target_kyous_count,
    getted_kyous_count,
    estimate_aggregate_task,
    finished_aggregate_task,
    loaded_kyous,

    // Computed
    start_date_str,
    end_date_str,

    // Business logic
    reload,
    abort,
    set_loading,

    // Template event handlers
    add_definition,
    delete_current_definition,
    floating_action_button_style,
    apply,
    download_kyous_json,
    onRequestedAddDnoteListQuery,
    onRequestedAddDnoteItem,
    onRequestedAddDnoteTrendGraph,
    onRequestedAddDnoteCorrelationGraph,
    increment_finished_aggregate_task,

    // Event relay objects
    crudRelayHandlers,
    errorMessageRelayHandlers,
} = useDnoteView({ props, emits })

defineExpose({ reload, abort, set_loading })
</script>
<style lang="css" scoped>
/* fill_heightのとき固定高さ（常時） */
.dnote_view {
    height: v-bind('fill_height ? app_content_height.toString().concat("px") : "auto"');
    overflow-y: v-bind('fill_height ? "hidden" : "visible"');
    /* KyouListViewの幅(rykv-view.vueの:width="400")とあわせる */
    min-width: 400px;
    /* 編集時(ApplicationConfigからの表示)は最小高さを確保する */
    min-height: v-bind('editable ? "500px" : "auto"');
}

/* fill_height_mode のときだけ flex column レイアウトを有効化 */
.fill_height_mode {
    display: flex;
    flex-direction: column;
}

.fill_height_mode :deep(.v-window) {
    flex: 1;
    min-height: 0;
}

/* v-window / v-window-item の overflow-x は常時上書き（既存動作維持） */
:deep(.v-window__container) {
    overflow-x: visible !important;
}

:deep(.v-window-item) {
    overflow-x: visible !important;
}

/* height: 100% は fill_height_mode のときだけ */
.fill_height_mode :deep(.v-window__container) {
    height: 100% !important;
}

.fill_height_mode :deep(.v-window-item) {
    height: 100% !important;
}

/* dnote-scroll-wrap: overflow-x は常時（既存動作維持） */
.dnote-scroll-wrap {
    overflow-x: auto;
}

/* flex column レイアウトと height:100% は fill_height_mode のときだけ */
.fill_height_mode .dnote-scroll-wrap {
    height: 100%;
    display: flex;
    flex-direction: column;
    /* リスト無し定義でも溢れ分を内部スクロール */
    overflow-y: auto;
}

.fill_height_mode :deep(.dnote_list_table_root) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
}

/* 集計リストが1つでもあるときは最低高さを確保する。
   この区画は flex-basis:0 なので、集計項目やグラフが縦を食い切ると
   高さ0＝完全に見えない状態になる。溢れたぶんは .dnote-scroll-wrap が
   縦スクロールで受ける。
   :has() で「リストが1つ以上あるとき」に限るのが肝 ―― 無条件に床を張ると、
   既定の定義（lists: []）でも200pxの空白が予約され、ダッシュボードの
   狭いパネルで余計なスクロールが出る */
.fill_height_mode :deep(.dnote_list_table_root:has(.dnote_list_view_root)) {
    min-height: 200px;
}

.fill_height_mode :deep(.dnote_list_table_row) {
    flex: 1;
    min-height: 0;
    align-items: stretch !important;
}

.fill_height_mode :deep(.dnote_list_view_root) {
    display: flex;
    flex-direction: column;
    min-height: 0;
    max-height: -webkit-fill-available;
}

/* v-virtual-scroll: fill_heightのとき残り高さをflex-growで占める */
:deep(.dnote_list_view) {
    height: v-bind('fill_height ? "0" : "50vh"') !important;
    flex-grow: v-bind('fill_height ? "1" : "0"') !important;
}

.fill_height_mode :deep(.dnote_list_view) {
    min-height: 0 !important;
}

.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading ? 'calc(100vw)' : '0px'");
    display: flex;
    flex-direction: column;
    align-items: center;
}

.overlay_message {
    text-align: center;
}

/* 編集時(ダイアログ表示)はボタン行をダイアログ下端に張り付ける */
.dnote_actions {
    display: block;
    flex: 0 0 auto;
    margin-top: auto;
    position: sticky;
    bottom: 0;
    z-index: 5;
    background-color: rgb(var(--v-theme-surface));
    padding: 8px;
}
</style>
