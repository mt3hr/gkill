<template>
    <!-- 普段使う rykv 画面。**列の中身は持たない**（rykv-columns-view）。
         ここが持つのは「編集できる rykv にだけあるもの」—— 画面切替メニュー、
         編集系のダイアログ群、FAB、ヘルプ。HTML 出力のラッパ（exported-rykv-view）は
         これらを1つも持たないので、同じ列の子を別のラッパがホストする形にしてある。
         **モードのフラグは渡さない。** 何ができるかはリテラルで書く -->
    <RykvColumnsView :application_config="application_config" :gkill_api="gkill_api"
        :app_title_bar_height="app_title_bar_height" :app_content_height="app_content_height"
        :app_content_width="app_content_width" :application_config_load_failed="application_config_load_failed"
        :column_state_instance_key="column_state_instance_key" :kyou_change_channel="kyou_change_channel"
        :enable_context_menu="true" :enable_add_column="true" :is_readonly_mi_check="false"
        :show_column_footer="true" :show_rep_name="true" :show_timeis_playing_end_button="true"
        :is_dnote_available="true" :is_map_available="true" :is_sidebar_available="true"
        :enable_keyboard_shortcuts="!is_hosted_in_dialog /* ポートでは枚数ぶん多重登録される */"
        v-on="columnsForwardHandlers"
        @requested_reload_application_config="emits('requested_reload_application_config')"
        @requested_export_kyou_html="show_export_kyou_html_dialog()"
        @requested_show_kftl_dialog="show_kftl_dialog()"
        @requested_show_save_clipboard_to_file_dialog="show_save_clipboard_to_file_dialog()">
        <template v-slot:app_bar_title>
            <div>
                <span>{{ i18n.global.t("RYKV_APP_TITLE") }}</span>
                <v-menu activator="parent">
                    <v-list>
                        <v-list-item :key="index" :value="index" v-for="page, index in page_list"
                            @click="navigate_to_page(page.page_name)">
                            <v-list-item-title>
                                {{ page.app_name }}</v-list-item-title>
                        </v-list-item>
                    </v-list>
                </v-menu>
            </div>
        </template>
        <template v-slot:app_bar_actions>
            <v-tooltip :text="i18n.global.t('TOOLTIP_HELP')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-help-circle-outline" @click="show_help_dialog()" />
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_SETTINGS')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-cog" :disabled="!application_config.is_loaded"
                        @click="emits('requested_show_application_config_dialog')" />
                </template>
            </v-tooltip>
        </template>
        <!-- 編集系のダイアログ群。**中継束は列から受け取る** ――
             保存した記録は再検索せず列へ差し込むので、ここの emit が
             列の中継束へ流れていなければ「追加した記録が一覧に出ない」になる -->
        <template v-slot:dialogs="{ crud_relay, all_columns_request_relay, querys, focused_column_index }">
            <AddKCDialog :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="true" :enable_dialog="true"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                ref="add_kc_dialog" />
            <AddTimeIsDialog :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="true" :enable_dialog="true"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                ref="add_timeis_dialog" />
            <AddLantanaDialog :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="true" :enable_dialog="true"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                ref="add_lantana_dialog" />
            <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="true" :enable_dialog="true"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                ref="add_urlog_dialog" />
            <AddMiDialog :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="true" :enable_dialog="true"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                ref="add_mi_dialog" />
            <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="true" :enable_dialog="true"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                ref="add_nlog_dialog" />
            <KFTLDialogHost :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :app_content_height="app_content_height" :enable_context_menu="true"
                :enable_dialog="true" :app_content_width="app_content_width"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                @saved_kyou_by_kftl="(last_added_request_time: Date) => emits('saved_kyou_by_kftl', last_added_request_time)"
                ref="kftl_dialog" />
            <mkflDialog :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :app_content_height="app_content_height" :enable_context_menu="true"
                :enable_dialog="true" :app_content_width="app_content_width"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                @saved_kyou_by_kftl="(last_added_request_time: Date) => emits('saved_kyou_by_kftl', last_added_request_time)"
                ref="mkfl_dialog" />
            <UploadFileDialog :app_content_height="app_content_height"
                :app_content_width="app_content_width" :application_config="application_config" :gkill_api="gkill_api"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                ref="upload_file_dialog" />
            <SaveClipboardToFileDialog :app_content_height="app_content_height"
                :app_content_width="app_content_width" :application_config="application_config" :gkill_api="gkill_api"
                v-on="{ ...crud_relay, ...all_columns_request_relay }"
                ref="save_clipboard_to_file_dialog" />
            <!--
                「HTML で出力」。列の条件はサイドバーが生成したまま渡す —— 削ると
                出力先のサイドバーが初めに生成した条件と食い違い、その列が空になる
            -->
            <ExportKyouHTMLDialog :application_config="application_config"
                :gkill_api="gkill_api" :columns="querys" :focused_column_index="focused_column_index"
                :screen="'rykv'"
                @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                ref="export_kyou_html_dialog" />
            <HelpDialog screen_name="rykv" ref="help_dialog" />
        </template>
        <!-- ポート(rudbeckia)の中ではFABを出さない。.position-fixed は position: fixed なので
             ダイアログを抜けて画面右下に居座り、ポート自身のFABと重なる。
             記録の追加はポートのFABから行う（同じダイアログを開く） -->
        <template v-slot:fab>
            <v-avatar v-if="!is_hosted_in_dialog" :style="floating_action_button_style()" color="primary"
                class="position-fixed">
                <v-menu transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" icon="mdi-plus" v-long-press="() => show_kftl_dialog()" variant="text"
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
                        <v-list-item @click="show_save_clipboard_to_file_dialog()">
                            <v-list-item-title>{{ i18n.global.t("SAVE_CLIPBOARD_TO_FILE_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                    </v-list>
                </v-menu>
            </v-avatar>
        </template>
    </RykvColumnsView>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { Kyou } from '@/classes/datas/kyou'
import AddKCDialog from '../dialogs/add-kc-dialog.vue'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import KFTLDialogHost from './kftl-dialog-host.vue'
import mkflDialog from '../dialogs/mkfl-dialog.vue'
import AddLantanaDialog from '../dialogs/add-lantana-dialog.vue'
import AddTimeIsDialog from '../dialogs/add-time-is-dialog.vue'
import AddUrlogDialog from '../dialogs/add-ur-log-dialog.vue'
import UploadFileDialog from '../dialogs/upload-file-dialog.vue'
import SaveClipboardToFileDialog from '../dialogs/save-clipboard-to-file-dialog.vue'
import ExportKyouHTMLDialog from '../dialogs/export-kyou-html-dialog.vue'
import HelpDialog from '../dialogs/help-dialog.vue'
import RykvColumnsView from './rykv-columns-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { RykvViewEmits } from './rykv-view-emits'
import type { RykvViewProps } from './rykv-view-props'
import { useRykvView } from '@/classes/use-rykv-view'

const props = defineProps<RykvViewProps>()
const emits = defineEmits<RykvViewEmits>()

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
    save_clipboard_to_file_dialog,
    export_kyou_html_dialog,
    help_dialog,

    // Computed
    page_list,

    // Template event handlers
    navigate_to_page,

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
    show_save_clipboard_to_file_dialog,
    show_export_kyou_html_dialog,
    show_help_dialog,
    floating_action_button_style,

    // Event relay objects
    columnsForwardHandlers,
} = useRykvView({ props, emits })
</script>
