<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ALLOCATE_REP_TITLE") }}</span>
                    <span>{{ account.user_id }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="onClickAddRep">{{ i18n.global.t("ADD_TITLE")
                    }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card class="allocate_rep_list_view_card">
            <table class="allocate_rep_list_view">
                <tr v-for="repository in repositories" :key="repository.id">
                    <td>
                        <v-checkbox :label="i18n.global.t('ENABLE_TITLE')" v-model="repository.is_enable" />
                    </td>
                    <td>
                        <v-checkbox :label="i18n.global.t('WRITE_TITLE')" v-model="repository.use_to_write" />
                    </td>
                    <td>
                        <v-checkbox :label="i18n.global.t('AUTO_ALLOCATE_ID_TITLE')"
                            v-model="repository.is_execute_idf_when_reload" />
                    </td>
                    <td>
                        <v-checkbox :label="i18n.global.t('WATCH_TARGET_FOR_UPDATE_REP_TITLE')"
                            v-model="repository.is_watch_target_for_update_rep" />
                    </td>
                    <td>
                        <v-select ::label="i18n.global.t('DEVICE_NAME_TITLE')" v-model="repository.device"
                            :items="devices" />
                    </td>
                    <td>
                        <v-select v-model="repository.type" readonly :items="rep_types"
                            :label="i18n.global.t('REP_TYPE_TITLE')" />
                    </td>
                    <td>
                        <v-text-field :width="600" :label="i18n.global.t('FILE_PATH_TITLE')"
                            v-model="repository.file" />
                    </td>
                    <td>
                        <v-btn dark color="secondary" @click="show_confirm_delete_rep_dialog(repository)">{{
                            i18n.global.t("DELETE_TITLE") }}</v-btn>
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ i18n.global.t('APPLY_TITLE') }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="onRequestedCloseDialog">{{
                        i18n.global.t('CANCEL_TITLE') }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddRepDialog :application_config="application_config" :gkill_api="gkill_api" :server_configs="server_configs"
            :account="account"
            v-on="addRepHandlers"
            ref="add_rep_dialog" />
        <ConfirmDeleteRepDialog :application_config="application_config" :gkill_api="gkill_api"
            :rep_id="delete_target_rep ? delete_target_rep.id : ''" :server_configs="server_configs"
            v-on="confirmDeleteRepHandlers"
            ref="confirm_delete_rep_dialog" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { AllocateRepViewEmits } from './allocate-rep-view-emits'
import type { AllocateRepViewProps } from './allocate-rep-view-props'
import AddRepDialog from '../dialogs/add-rep-dialog.vue'
import ConfirmDeleteRepDialog from '../dialogs/confirm-delete-rep-dialog.vue'
import { useAllocateRepView } from '@/classes/use-allocate-rep-view'

const props = defineProps<AllocateRepViewProps>()
const emits = defineEmits<AllocateRepViewEmits>()

const {
    // Template refs
    add_rep_dialog,
    confirm_delete_rep_dialog,

    // State
    delete_target_rep,
    repositories,
    rep_types,
    devices,

    // Business logic
    show_confirm_delete_rep_dialog,
    apply,

    // Template event handlers
    onClickAddRep,
    onRequestedCloseDialog,

    // Event relay objects
    addRepHandlers,
    confirmDeleteRepHandlers,
} = useAllocateRepView({ props, emits })
</script>

<style lang="css" scoped>
.allocate_rep_list_view_card {
    overflow-x: scroll;
}

.allocate_rep_list_view {
    width: max-content;
}
</style>
