<template>
    <v-card variant="flat">
        <div v-if="target_kyou" class="ryuu_views" :class="{ ryuu_editable_mode: editable }">
            <v-tabs v-if="ryuu_definitions.length > 1 || editable" v-model="current_definition_index" show-arrows
                :center-active="false">
                <v-tab v-for="(def, i) in ryuu_definitions" :key="i" :value="i">
                    {{ def.name }}
                </v-tab>
                <v-btn v-if="editable" icon="mdi-plus" size="small" variant="text" class="align-self-center ml-1"
                    @click="add_definition" :title="i18n.global.t('ADD_RYUU_DEFINITION_TITLE')" />
            </v-tabs>
            <v-row v-if="editable && ryuu_definitions.length > 0" class="pa-2 ma-0" align="center">
                <v-col class="pa-0 ma-0">
                    <v-text-field v-model="ryuu_definitions[current_definition_index].name"
                        :label="i18n.global.t('RYUU_DEFINITION_NAME_LABEL')" density="compact" hide-details />
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn v-if="ryuu_definitions.length > 1" icon="mdi-delete" size="small" variant="text"
                        @click="delete_current_definition" :title="i18n.global.t('DELETE_RYUU_DEFINITION_TITLE')" />
                </v-col>
            </v-row>

            <v-window v-model="current_definition_index" :touch="false">
                <v-window-item v-for="(def, i) in ryuu_definitions" :key="i" :value="i">
                    <RyuuItemView v-for="(query, qIdx) in def.queries" :key="query.id" v-model="def.queries[qIdx]"
                        :gkill_api="gkill_api" :application_config="application_config" :enable_dialog="true"
                        :enable_context_menu="true" :target_kyou="target_kyou" :abort_controller="abort_controler"
                        :find_kyou_query_default="find_kyou_query_default" :matched_kyous="matched_kyous"
                        :editable="editable"
                        @requested_move_related_kyou_query="(group_id: string, query_id: string, direction: 'up' | 'down') => onRequestedMoveRelatedKyouQuery(group_id, query_id, direction)"
                        @requested_delete_related_kyou_list_query="(id: string) => onRequestedDeleteRelatedKyouListQuery(id)"
                        v-on="{ ...ryuuListItemCrudRelayHandlers, ...ryuuListItemRequestHandlers, ...ryuuListItemFocusHandlers, ...rykvDialogHandler }"
                        ref="related_kyou_list_item_views" />
                </v-window-item>
            </v-window>

            <AddRyuuItemDialog :gkill_api="gkill_api" :application_config="application_config"
                @requested_add_related_kyou_query="(query: RelatedKyouQuery) => onRequestedAddRelatedKyouQuery(query)"
                @received_errors="(errors: GkillError[]) => onReceivedErrors(errors)"
                @received_messages="(messages: GkillMessage[]) => onReceivedMessages(messages)"
                ref="add_ryuu_item_dialog" />

            <v-avatar v-if="editable" :style="floatingActionButtonStyle()" color="primary" class="position-fixed-ryuu">
                <v-menu transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" icon="mdi-plus" variant="text" v-bind="props" @click="onAddButtonClick" />
                    </template>
                </v-menu>
            </v-avatar>

            <v-card-action v-if="editable" class="ryuu_actions">
                <v-row class="pa-0 ma-0">
                    <v-col cols="auto" class="pa-0 ma-0">
                        <v-btn dark @click="onApplyClick" color="primary">{{ i18n.global.t("APPLY_TITLE") }}</v-btn>
                    </v-col>
                    <v-spacer />
                    <v-col cols="auto" class="pa-0 ma-0">
                        <v-btn dark color="secondary" @click="onCancelClick">
                            {{ i18n.global.t("CANCEL_TITLE") }}
                        </v-btn>
                    </v-col>
                </v-row>
            </v-card-action>
        </div>
    </v-card>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import AddRyuuItemDialog from '../dialogs/add-ryuu-item-dialog.vue'
import RyuuItemView from './ryuu-item-view.vue'
import RelatedKyouQuery from '../../classes/dnote/related-kyou-query'
import type RyuuViewProps from './ryuu-view-props'
import type RyuuViewEmits from './ryuu-view-emits'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useRyuuView } from '@/classes/use-ryuu-view'

const model_value = defineModel<ApplicationConfig>()
const props = defineProps<RyuuViewProps>()
const emits = defineEmits<RyuuViewEmits>()

const {
    // Template refs
    add_ryuu_item_dialog,
    related_kyou_list_item_views,

    // State
    ryuu_definitions,
    current_definition_index,
    abort_controler,

    // Business logic
    add_definition,
    delete_current_definition,
    floatingActionButtonStyle,

    // Template event handlers
    onRequestedMoveRelatedKyouQuery,
    onRequestedDeleteRelatedKyouListQuery,
    onReceivedErrors,
    onReceivedMessages,
    onRequestedAddRelatedKyouQuery,
    onAddButtonClick,
    onApplyClick,
    onCancelClick,

    // Event relay objects
    ryuuListItemCrudRelayHandlers,
    ryuuListItemRequestHandlers,
    ryuuListItemFocusHandlers,
    rykvDialogHandler,
} = useRyuuView({ props, emits, model_value })
</script>

<style lang="css" scoped>
.ryuu_views {
    position: relative;
    width: -webkit-fill-available;
    min-width: 400px;
    /* 編集時(ApplicationConfigからの表示)の最小高さはアイテム一覧側で確保する */
    min-height: v-bind('editable ? "unset" : "20vh"');
}

/* 編集時(ダイアログ表示)はflex縦積みでダイアログいっぱいに広げる */
.ryuu_editable_mode {
    display: flex;
    flex-direction: column;
    flex: 1 1 auto;
}

/* アイテムテーブル領域: 最小高さ80px + ダイアログ拡大時の余白吸収 */
.ryuu_editable_mode :deep(.v-window) {
    flex: 1 0 auto;
    min-height: 80px;
}

/* 編集時(ダイアログ表示)はボタン行をダイアログ下端に張り付ける */
.ryuu_actions {
    display: block;
    flex: 0 0 auto;
    position: sticky;
    bottom: 0;
    z-index: 5;
    background-color: rgb(var(--v-theme-surface));
    padding: 8px;
}
</style>
