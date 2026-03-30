<template>
    <v-card class="dnote_view">
        <v-overlay v-model="is_loading" :content-class="'dnote_progress_overlay'" class="align-center justify-center"
            contained persistent>
            <v-progress-circular indeterminate color="primary" class="align-center justify-center" />
            <div v-if="getted_kyous_count !== target_kyous_count" class="align-center justify-center">
                <div class="align-center justify-center overlay_message">
                    {{ i18n.global.t('DNOTE_GETTING_DATA') }}
                </div>
                <div class="align-center justify-center overlay_message">
                    {{ getted_kyous_count }}/{{ target_kyous_count }}
                </div>
            </div>
            <div v-if="getted_kyous_count === target_kyous_count" class="align-center justify-center">
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
        <v-tabs v-if="dnote_definitions.length > 1 || editable" v-model="current_definition_index" show-arrows>
            <v-tab v-for="(def, i) in dnote_definitions" :key="i" :value="i">
                {{ def.name }}
            </v-tab>
            <v-tooltip :text="i18n.global.t('ADD_DNOTE_DEFINITION_TITLE')">
                <template v-slot:activator="{ props }">
                    <v-btn v-if="editable" v-bind="props" icon="mdi-plus" size="small" variant="text" class="align-self-center ml-1"
                        @click="add_definition" />
                </template>
            </v-tooltip>
        </v-tabs>
        <v-row v-if="editable && dnote_definitions.length > 0" class="pa-2 ma-0" align="center">
            <v-col class="pa-0 ma-0">
                <v-text-field v-model="dnote_definitions[current_definition_index].name"
                    :label="i18n.global.t('DNOTE_DEFINITION_NAME_LABEL')" density="compact" hide-details />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <v-tooltip :text="i18n.global.t('DELETE_DNOTE_DEFINITION_TITLE')">
                    <template v-slot:activator="{ props }">
                        <v-btn v-if="dnote_definitions.length > 1" v-bind="props" icon="mdi-delete" size="small" variant="text"
                            @click="delete_current_definition" />
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
                            <v-btn v-bind="props" :disabled="!loaded_kyous" icon="mdi-download-circle-outline" @click="download_kyous_json" />
                        </template>
                    </v-tooltip>
                </v-col>
            </v-row>
        </h1>
        <v-window v-model="current_definition_index">
            <v-window-item v-for="(def, i) in dnote_definitions" :key="i" :value="i" :eager="true">
                <DnoteItemTableView :application_config="application_config" :gkill_api="gkill_api" :editable="editable"
                    v-model="dnote_definitions[i].items"
                    v-on="{ ...crudRelayHandlers, ...focusClickRelayHandlers, ...rykvDialogHandler }"
                    @finish_a_aggregate_task="incrementFinishedAggregateTask" :ref="(el) => set_item_table_ref(i, el)" />
                <DnoteListTableView :application_config="application_config" :gkill_api="gkill_api" :editable="editable"
                    v-if="dnote_definitions[i].lists" v-model="dnote_definitions[i].lists"
                    v-on="{ ...crudRelayHandlers, ...focusClickRelayHandlers, ...rykvDialogHandler }"
                    @finish_a_aggregate_task="incrementFinishedAggregateTask" :ref="(el) => set_list_table_ref(i, el)" />
            </v-window-item>
        </v-window>
        <v-avatar v-if="editable" :style="floatingActionButtonStyle()" color="primary" class="position-fixed-dnote">
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
                </v-list>
            </v-menu>
        </v-avatar>
        <v-row v-if="editable" class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark @click="apply" color="primary">{{ i18n.global.t("APPLY_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ i18n.global.t("CANCEL_TITLE")
                }}</v-btn>
            </v-col>
        </v-row>
        <AddDnoteListDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorsMessagesRelayHandlers"
            @requested_add_dnote_list_query="(query: DnoteListQuery) => onRequestedAddDnoteListQuery(query)"
            ref="add_dnote_list_dialog" />
        <AddDnoteItemDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorsMessagesRelayHandlers"
            @requested_add_dnote_item="(item: DnoteItemData) => onRequestedAddDnoteItem(item)"
            ref="add_dnote_item_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type DnoteViewProps } from '@/pages/views/dnote-view-props'
import DnoteItemTableView from './dnote-item-table-view.vue'
import DnoteListTableView from './dnote-list-table-view.vue'
import AddDnoteListDialog from '../../pages/dialogs/add-dnote-list-dialog.vue'
import AddDnoteItemDialog from '../../pages/dialogs/add-dnote-item-dialog.vue'
import { type DnoteEmits } from '@/pages/views/dnote-emits'
import { useDnoteView } from '@/classes/use-dnote-view'
import type DnoteListQuery from "@/pages/views/dnote-list-query"
import type DnoteItem from "@/classes/dnote/dnote-item"
type DnoteItemData = DnoteItem

const props = defineProps<DnoteViewProps>()
const emits = defineEmits<DnoteEmits>()

const {
    // Template refs
    add_dnote_list_dialog,
    add_dnote_item_dialog,

    // View ref helpers
    set_item_table_ref,
    set_list_table_ref,

    // State
    dnote_definitions,
    current_definition_index,
    is_loading,
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

    // Template event handlers
    add_definition,
    delete_current_definition,
    floatingActionButtonStyle,
    apply,
    download_kyous_json,
    onRequestedAddDnoteListQuery,
    onRequestedAddDnoteItem,
    incrementFinishedAggregateTask,

    // Event relay objects
    crudRelayHandlers,
    focusClickRelayHandlers,
    rykvDialogHandler,
    errorsMessagesRelayHandlers,
} = useDnoteView({ props, emits })

defineExpose({ reload, abort })
</script>
<style lang="css" scoped>
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
</style>
