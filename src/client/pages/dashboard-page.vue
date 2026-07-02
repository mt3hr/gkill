<template>
    <div class="dashboard_view_wrap" ref="dashboard_root">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }"
                class="d-none d-md-flex" />
            <v-toolbar-title>
                <div>
                    <span>{{ i18n.global.t("DASHBOARD_PAGE_TITLE") }}</span>
                    <v-menu activator="parent">
                        <v-list>
                            <v-list-item :key="index" :value="index" v-for="page, index in page_list">
                                <v-list-item-title @click="navigateToPage(page.page_name)">
                                    {{ page.app_name }}
                                </v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-menu>
                </div>
            </v-toolbar-title>
            <v-spacer />
            <v-btn icon @click="go_prev_day()" variant="text">
                <v-icon>mdi-chevron-left</v-icon>
            </v-btn>
            <v-menu v-model="date_picker_menu_open" :close-on-content-click="false" location="bottom end"
                :z-index="3000">
                <template v-slot:activator="{ props: dateMenuProps }">
                    <v-btn variant="text" class="text-none" v-bind="dateMenuProps">
                        {{ date_label }}
                    </v-btn>
                </template>
                <v-date-picker v-model="date_picker_model"
                    @update:model-value="date_picker_menu_open = false" />
            </v-menu>
            <v-btn icon @click="go_next_day()" variant="text">
                <v-icon>mdi-chevron-right</v-icon>
            </v-btn>
            <v-divider vertical class="mx-1 d-none d-md-flex" />
            <v-btn icon="mdi-help-circle-outline" @click="help_dialog?.show()" />
            <v-tooltip :text="i18n.global.t('TOOLTIP_SETTINGS')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-cog" :disabled="!application_config.is_loaded"
                        @click="application_config_dialog?.show()" />
                </template>
            </v-tooltip>
        </v-app-bar>
        <v-main class="main pa-0 ma-0">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <div :class="is_loading ? 'hide' : 'show'" class="dashboard-content-scroll">
                <v-container fluid class="pa-0 ma-0 dashboard-container">
                    <v-row no-gutters class="pa-0 ma-0">
                        <v-col cols="12" md="6" class="pa-0 ma-0 dnote_view_wrap">
                            <DnoteView ref="dnote_view" :query="dnote_query" :checked_kyous="checked_kyous"
                                :app_content_height="panel_height" :app_content_width="app_content_width"
                                :application_config="application_config" :gkill_api="gkill_api" :editable="false"
                                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                                @requested_open_rykv_dialog="(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload)" />
                        </v-col>
                        <v-col cols="12" md="6" class="pa-0 ma-0">
                            <div class="gps-map-container">
                                <GPSLogMap :start_date="target_date_start" :end_date="target_date_end"
                                    :marker_time="target_date_start" :application_config="application_config"
                                    :gkill_api="gkill_api" :app_content_height="panel_height"
                                    @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                                    @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                                    ref="gps_log_map" />
                            </div>
                        </v-col>
                        <v-col cols="12" class="pa-0 ma-0">
                            <v-card class="mi-list-card pa-0 ma-0">
                                <v-card-title class="pa-2">{{ i18n.global.t("MI_APP_NAME") }}</v-card-title>
                                <KyouListView ref="mi_list_view" :query="mi_kyou_query" :matched_kyous="mi_kyous"
                                    :is_focused_list="true" :list_height="mi_kyous.length * 91 + 48" :kyou_height="91"
                                    :width="Math.max(0, app_content_width - 8)" :show_footer="true"
                                    :show_checkbox="false" :closable="false" :is_readonly_mi_check="false"
                                    :enable_context_menu="true" :enable_dialog="true" :show_content_only="false"
                                    :show_timeis_plaing_end_button="false" :is_show_doc_image_toggle_button="false"
                                    :is_show_arrow_button="false" :force_show_latest_kyou_info="true"
                                    :show_rep_name="false" :application_config="application_config"
                                    :gkill_api="gkill_api"
                                    @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                                    @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                                    @requested_open_rykv_dialog="(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload)" />
                            </v-card>
                        </v-col>
                    </v-row>
                </v-container>
            </div>
            <v-avatar :style="floatingActionButtonStyle()" color="primary" class="position-fixed">
                <v-menu transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" icon="mdi-plus" variant="text" v-long-press="() => show_kftl_dialog()"
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
            <AddKCDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="true" :enable_dialog="false"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="add_kc_dialog" />
            <AddTimeisDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="true" :enable_dialog="false"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="add_timeis_dialog" />
            <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="true" :enable_dialog="false"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="add_lantana_dialog" />
            <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="true" :enable_dialog="false"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="add_urlog_dialog" />
            <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="true" :enable_dialog="false"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="add_mi_dialog" />
            <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="true" :enable_dialog="false"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="add_nlog_dialog" />
            <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :app_content_height="app_content_height" :enable_context_menu="true"
                :enable_dialog="false" :app_content_width="app_content_width"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="kftl_dialog" />
            <mkflDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :app_content_height="app_content_height" :enable_context_menu="true"
                :enable_dialog="false" :app_content_width="app_content_width"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="mkfl_dialog" />
            <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="upload_file_dialog" />
            <SaveClipboardToFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                ref="save_clipboard_to_file_dialog" />
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :enable_context_menu="true" :enable_dialog="true" @closed="(id: string) => close_rykv_dialog(id)"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)" />
            <ConfirmLogoutDialog @requested_logout="(close_database: boolean) => logout(close_database)"
                ref="confirm_logout_dialog" />
            <HelpDialog screen_name="dashboard" ref="help_dialog" />
            <TutorialDialog :application_config="application_config" :gkill_api="gkill_api"
                ref="tutorial_dialog" />
            <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
                :app_content_height="app_content_height" :app_content_width="app_content_width"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                @requested_reload_application_config="load_application_config" ref="application_config_dialog" />
        </v-main>
        <div class="alert_container">
            <v-slide-y-transition group>
                <v-tooltip :text="(message.is_error ? 'エラーコード' : 'メッセージコード') + ':' + message.code"
                    v-for="message in messages" :key="message.id">
                    <template v-slot:activator="{ props }">
                        <v-alert v-bind="props" :color="message.is_error ? 'error' : undefined"
                            :role="message.is_error ? 'alert' : undefined" :closable="message.closable"
                            @click:close="close_message(message.id)">
                            {{ message.message }}
                        </v-alert>
                    </template>
                </v-tooltip>
            </v-slide-y-transition>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { ref, computed, watch, nextTick } from 'vue'
import { i18n } from '@/i18n'
import DnoteView from './views/dnote-view.vue'
import GPSLogMap from './views/gps-log-map.vue'
import KyouListView from './views/kyou-list-view.vue'
import RykvDialogHost from './views/rykv-dialog-host.vue'
import ConfirmLogoutDialog from './dialogs/confirm-logout-dialog.vue'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import HelpDialog from './dialogs/help-dialog.vue'
import TutorialDialog from './dialogs/tutorial-dialog.vue'
import AddKCDialog from './dialogs/add-kc-dialog.vue'
import AddTimeisDialog from './dialogs/add-timeis-dialog.vue'
import AddLantanaDialog from './dialogs/add-lantana-dialog.vue'
import AddUrlogDialog from './dialogs/add-urlog-dialog.vue'
import AddMiDialog from './dialogs/add-mi-dialog.vue'
import AddNlogDialog from './dialogs/add-nlog-dialog.vue'
import kftlDialog from './dialogs/kftl-dialog.vue'
import mkflDialog from './dialogs/mkfl-dialog.vue'
import UploadFileDialog from './dialogs/upload-file-dialog.vue'
import SaveClipboardToFileDialog from './dialogs/save-clipboard-to-file-dialog.vue'
import { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { useDashboardPage } from '@/classes/use-dashboard-page'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
const tutorial_dialog = ref<InstanceType<typeof TutorialDialog> | null>(null)
const dnote_view = ref<InstanceType<typeof DnoteView> | null>(null)
const gps_log_map = ref<InstanceType<typeof GPSLogMap> | null>(null)
const mi_list_view = ref<InstanceType<typeof KyouListView> | null>(null)
const application_config_dialog = ref<InstanceType<typeof ApplicationConfigDialog> | null>(null)
const date_picker_menu_open = ref(false)

const {
    // Template refs
    dashboard_root,
    confirm_logout_dialog,
    add_mi_dialog,
    add_nlog_dialog,
    add_lantana_dialog,
    add_timeis_dialog,
    add_urlog_dialog,
    kftl_dialog,
    add_kc_dialog,
    mkfl_dialog,
    upload_file_dialog,
    save_clipboard_to_file_dialog,

    // State
    is_loading,
    actual_height,
    app_title_bar_height,
    gkill_api,
    application_config,
    app_content_height,
    app_content_width,
    messages,
    selected_date,
    checked_kyous,
    mi_kyous,
    opened_dialogs,

    // Computed
    panel_height,
    page_list,
    target_date_start,
    target_date_end,
    date_label,
    date_picker_model,
    dnote_query,
    mi_kyou_query,

    // Methods
    write_errors,
    write_messages,
    close_message,
    navigateToPage,
    abort_all_fetches,
    clear_dashboard_datas,
    load_application_config,
    fetch_dnote_kyous,
    fetch_mi_kyous,
    go_prev_day,
    go_next_day,
    go_today: _go_today,
    floatingActionButtonStyle,
    show_confirm_logout_dialog: _show_confirm_logout_dialog,
    show_kftl_dialog,
    show_add_kc_dialog,
    show_mkfl_dialog,
    show_timeis_dialog,
    show_mi_dialog,
    show_nlog_dialog,
    show_lantana_dialog,
    show_urlog_dialog,
    show_upload_file_dialog,
    show_save_clipboard_to_file_dialog: _show_save_clipboard_to_file_dialog,
    logout,
    open_rykv_dialog,
    close_rykv_dialog,
} = useDashboardPage()

// 日付変更時: DnoteView・KyouListView それぞれのローディングを使う
let fetch_id = 0
async function fetch_for_date(): Promise<void> {
    const my_id = ++fetch_id
    abort_all_fetches()
    clear_dashboard_datas()
    dnote_view.value?.set_loading(true)
    mi_list_view.value?.set_loading(true)
    await nextTick()
    await Promise.all([
        fetch_mi_kyous().then(() => {
            if (my_id === fetch_id) mi_list_view.value?.set_loading(false)
        }),
        fetch_dnote_kyous().then((kyous) => {
            if (my_id === fetch_id) dnote_view.value?.reload(kyous, dnote_query.value)
        }),
    ])
}

// 日付変更時にデータを再取得
watch(selected_date, () => {
    fetch_for_date()
})

watch(application_config, (config) => {
    if (config.is_loaded && config.show_tutorial_on_startup) {
        nextTick(() => tutorial_dialog.value?.show())
    }
})

const dnote_view_element_height = ref(0)
watch(() => application_config.value.is_loaded, async () => {
    await nextTick(() => { })
    dnote_view_element_height.value = document.getElementsByClassName("dnote_view")[0]?.clientHeight ?? 0
    await nextTick(() => { })
    gps_log_map.value?.centering()
    // 初期データ取得: application_config ロード完了後にデータを取得する
    // is_loading は use-dashboard-page.ts 側で false にしない（fetch完了後に隠す）
    await Promise.all([
        fetch_mi_kyous().then(() => mi_list_view.value?.set_loading(false)),
        fetch_dnote_kyous().then((kyous) => dnote_view.value?.reload(kyous, dnote_query.value)),
    ])
    is_loading.value = false
})

// CSS v-bind values for map height (Google Maps requires explicit pixel heights)
const map_height_px = computed((): string => {
    return `${dnote_view_element_height.value - 32 - 54}px`
})
const mi_list_height_px = computed((): string => {
    const mi_list_height = app_content_height.value - dnote_view_element_height.value - 44 - 48 // 44px is the height of the MI list title
    if (mi_list_height == 0) {
        return "200px"
    }
    return `${mi_list_height}px`
})
const app_content_height_px = computed(() => `${app_content_height.value}px`)
</script>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading == true ? 'calc(100vw)' : '0px'");
}

.dashboard_view_wrap {
    overflow-x: hidden;
    max-width: 100vw;
    width: 100%;
}

.main {
    padding-top: 50px !important;
}

.dnote_view_wrap,
map-container {
    max-height: fit-content;
}

.dashboard-content-scroll {
    height: v-bind(app_content_height_px);
    overflow-y: auto;
    overflow-x: hidden;
}

.dashboard-container {
    max-width: 100%;
    width: 100%;
    box-sizing: border-box;
    overflow-x: hidden;
}

.gps-map-container {
    width: 100%;
    overflow-x: unset;
}

.gps-map-container :deep(.gps_log_map_wrap) {
    width: 100% !important;
    height: auto !important;
}

.gps-map-container :deep(.map_container) {
    flex: none !important;
    height: v-bind(map_height_px) !important;
}

.gps-map-container :deep(.googlemap) {
    width: 100% !important;
    height: v-bind(map_height_px) !important;
}

::v-deep(.kyou_list_view_card) {
    overflow-y: scroll !important;
    min-height: v-bind(mi_list_height_px);
    max-height: v-bind(mi_list_height_px);
    height: v-bind(mi_list_height_px);
}

/* DnoteViewのテーブルが横はみ出しを引き起こす場合にクリップ */
:deep(.dnote_view) {
    max-width: 100%;
    overflow-x: hidden;
}

:deep(.dnote_view .dnote-scroll-wrap) {
    overflow-x: auto;
}

:deep(.dashboard-content-scroll .dnote_list_table_root .dnote_list_view) {
    height: 30vh !important;
    max-height: 30vh !important;
    min-height: 30vh !important;
    overflow-y: scroll;
}
</style>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}

.hide {
    opacity: 0;
}

.show {
    opacity: 1;
}
</style>
