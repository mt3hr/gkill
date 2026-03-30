<template>
    <v-card>
        <div v-if="target_kyou" class="ryuu_views">
            <v-tabs v-if="ryuu_definitions.length > 1 || editable" v-model="current_definition_index" show-arrows>
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
                        @click="delete_current_definition"
                        :title="i18n.global.t('DELETE_RYUU_DEFINITION_TITLE')" />
                </v-col>
            </v-row>

            <v-window v-model="current_definition_index">
                <v-window-item v-for="(def, i) in ryuu_definitions" :key="i" :value="i">
                    <RyuuItemView v-for="(query, qIdx) in def.queries" :key="query.id"
                        v-model="def.queries[qIdx]" :gkill_api="gkill_api" :application_config="application_config"
                        :enable_dialog="true" :enable_context_menu="true" :target_kyou="target_kyou"
                        :abort_controller="abort_controler" :find_kyou_query_default="find_kyou_query_default"
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
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :enable_context_menu="true" :enable_dialog="true"
                @closed="(id: string) => onDialogHostClosed(id)"
                v-on="{ ...ryuuListItemCrudRelayHandlers, ...ryuuListItemRequestHandlers, ...ryuuListItemFocusHandlers, ...rykvDialogHandler }" />

            <v-avatar v-if="editable" :style="floatingActionButtonStyle()" color="primary" class="position-fixed-ryuu">
                <v-menu transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" icon="mdi-plus" variant="text" v-bind="props"
                            @click="onAddButtonClick" />
                    </template>
                </v-menu>
            </v-avatar>

            <v-row v-if="editable" class="pa-0 ma-0">
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
import RykvDialogHost from './rykv-dialog-host.vue'
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
    opened_dialogs,
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
    onDialogHostClosed,
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
    min-height: 20vh;
}
</style>
