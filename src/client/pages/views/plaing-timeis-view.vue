<template>
    <div class="plaing_timeis_view_wrap">
        <KyouListView :kyou_height="180" :width="app_content_width" :list_height="kyou_list_view_height"
            :show_timeis_plaing_end_button="true" :application_config="application_config" :gkill_api="gkill_api"
            :matched_kyous="match_kyous_list" :query="query" :is_focused_list="true"
            :closable="false" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            :is_readonly_mi_check="false" :show_checkbox="true" :show_footer="true" :show_content_only="true"
            :show_rep_name="true" :force_show_latest_kyou_info="true" :is_show_doc_image_toggle_button="false"
            :is_show_arrow_button="false"
            v-on="{ ...crudRelayHandlers, ...reloadListRequestHandlers, ...rykvDialogHandler }"
            @requested_search="search(false)"
            ref="kyou_list_views" />
        <AddKCDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers"
            ref="add_kc_dialog" />
        <AddTimeisDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog"
            v-on="{ ...crudRelayHandlers, ...dialogReloadRequestHandlers }"
            ref="add_timeis_dialog" />
        <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog"
            v-on="{ ...crudRelayHandlers, ...dialogReloadRequestHandlers }"
            ref="add_lantana_dialog" />
        <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog"
            v-on="{ ...crudRelayHandlers, ...dialogReloadRequestHandlers }"
            ref="add_urlog_dialog" />
        <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog"
            v-on="{ ...crudRelayHandlers, ...dialogReloadRequestHandlers }"
            ref="add_mi_dialog" />
        <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog"
            v-on="{ ...crudRelayHandlers, ...dialogReloadRequestHandlers }"
            ref="add_nlog_dialog" />
        <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :kyou="new Kyou()" :app_content_height="app_content_height"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            :app_content_width="app_content_width"
            v-on="{ ...crudRelayHandlers, ...dialogReloadRequestHandlers }"
            ref="kftl_dialog" />
        <mkflDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :kyou="new Kyou()" :app_content_height="app_content_height"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            :app_content_width="app_content_width"
            v-on="{ ...crudRelayHandlers, ...dialogReloadRequestHandlers }"
            ref="mkfl_dialog" />
        <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            v-on="crudRelayHandlers"
            ref="upload_file_dialog" />
        <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @closed="(...id: any[]) => close_rykv_dialog(id[0] as string)"
            v-on="{ ...crudRelayHandlers, ...reloadListRequestHandlers, ...rykvDialogHandler }" />
        <v-avatar :style="floatingActionButtonStyle()" color="primary" class="position-fixed">
            <v-menu :style="add_kyou_menu_style" transition="slide-x-transition">
                <template v-slot:activator="{ props }">
                    <v-btn color="white" v-long-press="() => show_kftl_dialog()" icon="mdi-plus" variant="text"
                        v-bind="props" />
                </template>
                <v-list>
                    <v-list-item @click="show_kftl_dialog()">
                        <v-list-item-title>{{ i18n.global.t("KFTL_APP_NAME") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_mkfl_dialog()">
                        <v-list-item-title>{{ i18n.global.t("MKFL_APP_NAME") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_add_kc_dialog()">
                        <v-list-item-title>{{ i18n.global.t("KC_APP_NAME") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_urlog_dialog()">
                        <v-list-item-title>{{ i18n.global.t("URLOG_APP_NAME") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_timeis_dialog()">
                        <v-list-item-title>{{ i18n.global.t("TIMEIS_APP_NAME") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_mi_dialog()">
                        <v-list-item-title>{{ i18n.global.t("MI_APP_NAME") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_nlog_dialog()">
                        <v-list-item-title>{{ i18n.global.t("NLOG_APP_NAME") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_lantana_dialog()">
                        <v-list-item-title>{{ i18n.global.t("LANTANA_APP_NAME") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_upload_file_dialog()">
                        <v-list-item-title>{{ i18n.global.t("UPLOAD_APP_NAME") }}</v-list-item-title>
                    </v-list-item>
                </v-list>
            </v-menu>
        </v-avatar>
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { Kyou } from '@/classes/datas/kyou'
import AddKCDialog from '../dialogs/add-kc-dialog.vue'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import KyouListView from './kyou-list-view.vue'
import kftlDialog from '../dialogs/kftl-dialog.vue'
import mkflDialog from '../dialogs/mkfl-dialog.vue'
import AddLantanaDialog from '../dialogs/add-lantana-dialog.vue'
import AddTimeisDialog from '../dialogs/add-timeis-dialog.vue'
import AddUrlogDialog from '../dialogs/add-urlog-dialog.vue'
import UploadFileDialog from '../dialogs/upload-file-dialog.vue'
import type { PlaingTimeIsViewProps } from './plaing-timeis-view-props'
import type { PlaingTimeIsViewEmits } from './plaing-timeis-emits'
import RykvDialogHost from './rykv-dialog-host.vue'
import { usePlaingTimeisView } from '@/classes/use-plaing-timeis-view'

const props = defineProps<PlaingTimeIsViewProps>()
const emits = defineEmits<PlaingTimeIsViewEmits>()

const {
    // Template refs
    add_mi_dialog,
    add_nlog_dialog,
    add_lantana_dialog,
    add_timeis_dialog,
    add_urlog_dialog,
    kftl_dialog,
    add_kc_dialog,
    mkfl_dialog,
    upload_file_dialog,
    kyou_list_views,

    // State
    enable_context_menu,
    enable_dialog,
    opened_dialogs,
    query,
    match_kyous_list,

    // Computed
    kyou_list_view_height,
    add_kyou_menu_style,

    // Business logic
    reload_list,
    search,
    set_last_added_request_time,
    close_rykv_dialog,

    // Dialog show methods
    show_kftl_dialog,
    show_mkfl_dialog,
    show_add_kc_dialog,
    show_urlog_dialog,
    show_timeis_dialog,
    show_mi_dialog,
    show_nlog_dialog,
    show_lantana_dialog,
    show_upload_file_dialog,
    floatingActionButtonStyle,

    // Event relay objects
    crudRelayHandlers,
    reloadListRequestHandlers,
    dialogReloadRequestHandlers,
    rykvDialogHandler,
} = usePlaingTimeisView({ props, emits })

defineExpose({ reload_list, set_last_added_request_time })
</script>
<style lang="css" scoped>
.plaing_timeis_view_table {
    padding-top: 0px;
}

.kyou_detail_view .kyou_image {
    width: -webkit-fill-available !important;
    height: -webkit-fill-available !important;
    max-width: -webkit-fill-available !important;
    max-height: 100vh !important;
    object-fit: contain;
}

.kyou_detail_view .kyou_video {
    width: -webkit-fill-available !important;
    height: -webkit-fill-available !important;
    max-width: -webkit-fill-available !important;
    max-height: 100vh !important;
    object-fit: contain;
}

.plaing_timeis_view_wrap {
    overflow-y: hidden;
}
</style>
