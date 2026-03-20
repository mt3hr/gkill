<template>
    <div class="saihate_view_wrap" ref="saihate_root">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }" />
            <v-toolbar-title>{{ i18n.global.t("SAIHATE_PAGE_TITLE") }}</v-toolbar-title>
            <v-spacer />
            <v-tooltip :text="i18n.global.t('TOOLTIP_RELOAD')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon @click="() => reload_repositories(false)" v-long-press="() => reload_repositories(true)"
                        variant="text">
                        <v-icon>mdi-reload</v-icon>
                    </v-btn>
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_HELP')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-help-circle-outline" @click="help_dialog?.show()" />
                </template>
            </v-tooltip>
            <v-btn dark @click="() => show_confirm_logout_dialog(false)"
                v-long-press="() => show_confirm_logout_dialog(true)">
                {{ i18n.global.t("LOGOUT_TITLE") }}
            </v-btn>
        </v-app-bar>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
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
            <AddKCDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="add_kc_dialog" />
            <AddTimeisDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="add_timeis_dialog" />
            <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="add_lantana_dialog" />
            <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="add_urlog_dialog" />
            <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="add_mi_dialog" />
            <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="add_nlog_dialog" />
            <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :app_content_height="app_content_height" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :app_content_width="app_content_width"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="kftl_dialog" />
            <mkflDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :app_content_height="app_content_height" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :app_content_width="app_content_width"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="mkfl_dialog" />
            <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                ref="upload_file_dialog" />
            <ConfirmLogoutDialog @requested_logout="(close_database: boolean) => logout(close_database)"
                ref="confirm_logout_dialog" />
            <HelpDialog screen_name="saihate" ref="help_dialog" />
            <TutorialDialog :application_config="application_config" :gkill_api="gkill_api"
                ref="tutorial_dialog" />
        </v-main>
        <div class="alert_container">
            <v-slide-y-transition group>
                <v-tooltip :text="(message.is_error ? 'エラーコード' : 'メッセージコード') + ':' + message.code"
                    v-for="message in messages" :key="message.id">
                    <template v-slot:activator="{ props }">
                        <v-alert v-bind="props" :color="message.is_error ? 'error' : undefined"
                            :closable="message.closable" @click:close="close_message(message.id)">
                            {{ message.message }}
                        </v-alert>
                    </template>
                </v-tooltip>
            </v-slide-y-transition>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { ref, watch, nextTick } from 'vue'
import { i18n } from '@/i18n'
import { Kyou } from '@/classes/datas/kyou'
import AddKCDialog from './dialogs/add-kc-dialog.vue'
import AddTimeisDialog from './dialogs/add-timeis-dialog.vue'
import AddLantanaDialog from './dialogs/add-lantana-dialog.vue'
import AddUrlogDialog from './dialogs/add-urlog-dialog.vue'
import AddMiDialog from './dialogs/add-mi-dialog.vue'
import AddNlogDialog from './dialogs/add-nlog-dialog.vue'
import kftlDialog from './dialogs/kftl-dialog.vue'
import mkflDialog from './dialogs/mkfl-dialog.vue'
import UploadFileDialog from './dialogs/upload-file-dialog.vue'
import ConfirmLogoutDialog from './dialogs/confirm-logout-dialog.vue'
import HelpDialog from './dialogs/help-dialog.vue'
import TutorialDialog from './dialogs/tutorial-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useSaihatePage } from '@/classes/use-saihate-page'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
const tutorial_dialog = ref<InstanceType<typeof TutorialDialog> | null>(null)

const {
    // Template refs
    saihate_root,
    add_mi_dialog,
    add_nlog_dialog,
    add_lantana_dialog,
    add_timeis_dialog,
    add_urlog_dialog,
    kftl_dialog,
    add_kc_dialog,
    mkfl_dialog,
    upload_file_dialog,
    confirm_logout_dialog,

    // State
    enable_context_menu,
    enable_dialog,
    actual_height,
    app_title_bar_height,
    gkill_api,
    application_config,
    app_content_height,
    app_content_width,
    is_loading,
    messages,
    add_kyou_menu_style,

    // Methods
    write_errors,
    write_messages,
    close_message,
    floatingActionButtonStyle,

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
    show_confirm_logout_dialog,

    // Reload
    reload_repositories,

    // Logout
    logout,
} = useSaihatePage()

watch(application_config, (config) => {
    if (config.is_loaded && config.show_tutorial_on_startup) {
        nextTick(() => tutorial_dialog.value?.show())
    }
})
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
