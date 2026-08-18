<template>
    <div class="rudbeckia_view_wrap" ref="rudbeckia_root">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }" />
            <v-toolbar-title>
                <div>
                    <span>{{ i18n.global.t("RUDBECKIA_PAGE_TITLE") }}</span>
                    <!-- さいはてと違い、ここからページ遷移できる -->
                    <v-menu activator="parent">
                        <v-list>
                            <v-list-item :key="index" :value="index" v-for="page, index in page_list">
                                <v-list-item-title @click="navigate_to_page(page.page_name)">
                                    {{ page.app_name }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-menu>
                </div>
            </v-toolbar-title>
            <v-spacer />
            <v-tooltip :text="i18n.global.t('TOOLTIP_HELP')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-help-circle-outline" @click="help_dialog?.show()" />
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_SETTINGS')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-cog" :disabled="!application_config.is_loaded"
                        @click="show_application_config_dialog()" />
                </template>
            </v-tooltip>
        </v-app-bar>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <!-- 設定が取れないと is_loaded が立たず、ホストした画面の初期化が一度も走らない。
                         スピナーのままにすると永久に固まるので、再試行の導線を出す -->
                    <div v-if="application_config_load_failed" class="text-center">
                        <div class="mb-2">{{ i18n.global.t('FAILED_GET_APPLICATION_CONFIG_MESSAGE') }}</div>
                        <v-btn color="primary" @click="load_application_config()">
                            {{ i18n.global.t('RELOAD_TITLE') }}
                        </v-btn>
                    </div>
                    <v-progress-circular v-else indeterminate color="primary" />
                </v-overlay>
            </div>
            <!-- 画面ウィンドウより前に出す。覆われると唯一の操作導線が押せなくなる -->
            <v-avatar :style="floating_action_button_style()" color="primary" class="position-fixed-rudbeckia">
                <v-menu transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" v-long-press="() => show_kftl_dialog()" icon="mdi-plus" variant="text"
                            v-bind="props" />
                    </template>
                    <v-list class="gkill_context_menu_list">
                        <!-- 「画面」と「記録」で同じ名前（タスク）が並ぶので、見出しで区別する -->
                        <v-list-subheader>{{ i18n.global.t("RUDBECKIA_SCREEN_SECTION_TITLE") }}</v-list-subheader>
                        <v-list-item v-for="screen in screen_list" :key="screen.page_name"
                            @click="open_page_dialog(screen.page_name as RudbeckiaPageKind)">
                            <v-list-item-title>{{ screen.app_name }}</v-list-item-title>
                        </v-list-item>
                        <v-divider class="my-1" />
                        <v-list-subheader>{{ i18n.global.t("RUDBECKIA_RECORD_SECTION_TITLE") }}</v-list-subheader>
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
                        <v-list-item @click="show_save_clipboard_to_file_dialog()">
                            <v-list-item-title>{{ i18n.global.t("SAVE_CLIPBOARD_TO_FILE_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                    </v-list>
                </v-menu>
            </v-avatar>
            <!-- 画面ウィンドウ。ページ直下にフラットに置く。
                 ホストしたビューの中へ入れ子にすると createLayout が連鎖して
                 rootZIndex が段ごとに100下がる -->
            <RudbeckiaPageDialogHost :application_config="application_config" :gkill_api="gkill_api"
                :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config_load_failed="application_config_load_failed"
                :kyou_change_bus="kyou_change_bus"
                v-on="rudbeckiaKyouHandlers"
                @requested_open_page="open_page_dialog"
                @requested_navigate_page="navigate_to_page"
                @requested_show_application_config_dialog="show_application_config_dialog"
                @requested_reload_application_config="load_application_config"
                @saved_kyou_by_kftl="onSavedKyouByKftl"
                ref="page_dialog_host" />
            <AddKCDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="rudbeckiaKyouHandlers" ref="add_kc_dialog" />
            <AddTimeIsDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="rudbeckiaKyouHandlers" ref="add_timeis_dialog" />
            <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="rudbeckiaKyouHandlers" ref="add_lantana_dialog" />
            <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="rudbeckiaKyouHandlers" ref="add_urlog_dialog" />
            <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="rudbeckiaKyouHandlers" ref="add_mi_dialog" />
            <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="rudbeckiaKyouHandlers" ref="add_nlog_dialog" />
            <KFTLDialogHost :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :app_content_height="app_content_height" :app_content_width="app_content_width"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="rudbeckiaKyouHandlers"
                @saved_kyou_by_kftl="onSavedKyouByKftl" ref="kftl_dialog" />
            <mkflDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :app_content_height="app_content_height" :app_content_width="app_content_width"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="rudbeckiaKyouHandlers"
                @saved_kyou_by_kftl="onSavedKyouByKftl" ref="mkfl_dialog" />
            <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api"
                v-on="rudbeckiaKyouHandlers" ref="upload_file_dialog" />
            <SaveClipboardToFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api"
                v-on="rudbeckiaKyouHandlers" ref="save_clipboard_to_file_dialog" />
            <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
                :app_content_height="app_content_height" :app_content_width="app_content_width"
                @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
                @requested_reload_application_config="load_application_config" ref="application_config_dialog" />
            <HelpDialog screen_name="rudbeckia" ref="help_dialog" />
            <TutorialDialog :application_config="application_config" :gkill_api="gkill_api" ref="tutorial_dialog" />
        </v-main>
        <div class="alert_container" role="status" aria-live="polite">
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
import { ref } from 'vue'
import { useTutorialOnStartup } from '@/classes/use-tutorial-on-startup'
import { i18n } from '@/i18n'
import { Kyou } from '@/classes/datas/kyou'
import AddKCDialog from './dialogs/add-kc-dialog.vue'
import AddTimeIsDialog from './dialogs/add-time-is-dialog.vue'
import AddLantanaDialog from './dialogs/add-lantana-dialog.vue'
import AddUrlogDialog from './dialogs/add-ur-log-dialog.vue'
import AddMiDialog from './dialogs/add-mi-dialog.vue'
import AddNlogDialog from './dialogs/add-nlog-dialog.vue'
import KFTLDialogHost from './views/kftl-dialog-host.vue'
import mkflDialog from './dialogs/mkfl-dialog.vue'
import UploadFileDialog from './dialogs/upload-file-dialog.vue'
import SaveClipboardToFileDialog from './dialogs/save-clipboard-to-file-dialog.vue'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import HelpDialog from './dialogs/help-dialog.vue'
import TutorialDialog from './dialogs/tutorial-dialog.vue'
import RudbeckiaPageDialogHost from './views/rudbeckia-page-dialog-host.vue'
import type { RudbeckiaPageKind } from './views/rudbeckia-page-kind'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useRudbeckiaPage } from '@/classes/use-rudbeckia-page'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
const tutorial_dialog = ref<InstanceType<typeof TutorialDialog> | null>(null)

const {
    // Template refs
    rudbeckia_root,
    page_dialog_host,
    application_config_dialog,
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
    enable_context_menu,
    enable_dialog,
    actual_height,
    app_title_bar_height,
    gkill_api,
    application_config,
    application_config_load_failed,
    app_content_height,
    app_content_width,
    is_loading,
    messages,

    // Computed
    kyou_change_bus,
    page_list,
    screen_list,

    // Methods
    write_errors,
    write_messages,
    close_message,
    load_application_config,
    open_page_dialog,
    navigate_to_page,
    floating_action_button_style,

    // Dialog show methods
    show_application_config_dialog,
    show_kftl_dialog,
    show_mkfl_dialog,
    show_add_kc_dialog,
    show_urlog_dialog,
    show_timeis_dialog,
    show_mi_dialog,
    show_nlog_dialog,
    show_lantana_dialog,
    show_upload_file_dialog,
    show_save_clipboard_to_file_dialog,


    // Event relay objects
    rudbeckiaKyouHandlers,
    onSavedKyouByKftl,
} = useRudbeckiaPage()

useTutorialOnStartup(application_config, tutorial_dialog)
</script>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading ? 'calc(100vw)' : '0px'");
}
</style>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
