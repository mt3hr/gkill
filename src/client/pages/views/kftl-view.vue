<template>
    <v-card class="kftl_view">
        <!-- タブ列はタイトル行に同居させる。別の行にするとテキストエリアの縦が
             タブバーのぶんだけ削られる。
             タブは v-window を使わない ―― 非表示の textarea は clientWidth が 0 になり、
             行ラベルの行数計算が NaN に落ちる（kftl-statement-line.ts）ので、
             アクティブなタブ1枚だけを描画する -->
        <v-card-title class="kftl_title">
            <v-row class="flex-nowrap align-center pa-0 ma-0">
                <v-col class="pa-0 ma-0 kftl_tab_col">
                    <v-tabs v-model="active_tab_id_model" :height="tab_bar_height" density="compact" show-arrows
                        :center-active="false">
                        <v-tab v-for="(tab, index) in tabs" :key="tab.id" :value="tab.id" class="kftl_tab">
                            <span class="kftl_tab_label">{{ tab_label(tab, index) }}</span>
                            <!-- v-tab は button を描画するので、閉じるは v-btn ではなく v-icon にする（button のネストを避ける）。
                                 .stop が無いとタブのアクティブ化も一緒に起きる -->
                            <v-icon size="x-small" icon="mdi-close" class="ml-1 kftl_tab_close" role="button"
                                :aria-label="i18n.global.t('KFTL_TAB_CLOSE_TITLE')"
                                @click.stop.prevent="request_close_tab(tab.id)" />
                        </v-tab>
                        <v-tooltip :text="i18n.global.t('KFTL_TAB_ADD_TITLE')">
                            <template v-slot:activator="{ props }">
                                <v-btn v-bind="props" icon="mdi-plus" size="small" variant="text"
                                    class="align-self-center ml-1 kftl_tab_add" :disabled="is_tab_locked"
                                    @click="add_tab" />
                            </template>
                        </v-tooltip>
                    </v-tabs>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0 ml-2">
                    <!-- テンプレートは新しいタブを作るので、送信の確認往復中は押せないようにする -->
                    <v-btn dark color="primary" @click="show_kftl_template_dialog"
                        :disabled="is_requested_submit || is_tab_locked">{{
                            i18n.global.t("KFTL_TEMPLATE_TITLE") }}</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0 ml-2">
                    <v-btn dark color="primary" @click="submit" :disabled="is_requested_submit">{{
                        i18n.global.t("SAVE_TITLE")
                    }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <table class="kftl_input">
            <tbody>
                <tr>
                    <td>
                        <div class="kftl_line_label line_label_wrap" ref="kftl_line_label_wrap">
                            <KFTLLineLabel v-for="(line_label_data, index) in line_label_datas"
                                :key="index" :application_config="application_config"
                                :gkill_api="gkill_api" :line_label_data="line_label_data"
                                :style="line_label_styles[index]" />
                        </div>
                    </td>
                    <td>
                        <div class="kftl_text_area_wrap">
                            <textarea :id="text_area_element_id" class="kftl_text_area" ref="kftl_text_area"
                                v-model="text_area_content" :readonly="is_requested_submit"
                                @scroll="update_line_labels" @input="onTextAreaInput"></textarea>
                        </div>
                    </td>
                </tr>
            </tbody>
        </table>
        <KFTLTemplateDialog :application_config="application_config" :gkill_api="gkill_api"
            :template="application_config.kftl_template_struct"
            v-on="errorMessageRelayHandlers"
            @clicked_template_element_leaf="paste_template" ref="kftl_template_dialog" />
    </v-card>
    <Teleport to="body" v-if="show_confirm_unknown_tag_dialog">
        <div class="gkill-float-scrim" :class="confirm_dialog_ui.isTransparent.value ? 'is-transparent' : ''" />
        <div :ref="confirm_dialog_ui.containerRef" :style="confirm_dialog_ui.fixedStyle.value"
            class="gkill-floating-dialog"
            :class="confirm_dialog_ui.isTransparent.value ? 'is-transparent' : ''">
            <div class="gkill-floating-dialog__header pa-0 ma-0"
                @mousedown="confirm_dialog_ui.onHeaderPointerDown"
                @touchstart="confirm_dialog_ui.onHeaderPointerDown">
                <div class="gkill-floating-dialog__title"></div>
                <div class="gkill-floating-dialog__spacer"></div>
                <v-checkbox v-model="confirm_dialog_ui.isTransparent.value" color="white" size="small" variant="flat"
                    :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="cancel_submit()" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>
            <div class="gkill-floating-dialog__body">
                <v-card variant="flat" class="pa-2">
                    <v-card-title>
                        <v-row class="pa-0 ma-0">
                            <v-col cols="auto" class="pa-0 ma-0">
                                <span>{{ i18n.global.t("CONFIRM_UNKNOWN_TAG_TITLE") }}</span>
                            </v-col>
                        </v-row>
                    </v-card-title>
                    <v-card-text>
                        {{ i18n.global.t("CONFIRM_UNKNOWN_TAG_MESSAGE") }}
                        <v-list density="compact">
                            <v-list-item v-for="tag in unknown_tags" :key="tag">
                                <v-list-item-title>{{ tag }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-card-text>
                    <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
                        <v-col cols="auto" class="pa-0 ma-0">
                            <v-btn dark color="primary" @click="confirm_submit()"
                                :disabled="is_requested_submit">{{ i18n.global.t("SAVE_TITLE")
                                }}</v-btn>
                        </v-col>
                        <v-col cols="auto" class="pa-0 ma-0">
                            <v-btn @click="cancel_submit()">{{ i18n.global.t("CANCEL_TITLE") }}</v-btn>
                        </v-col>
                        <v-spacer />
                    </v-row>
                </v-card>
            </div>
        </div>
    </Teleport>
    <!-- タグの確認を通した後に出る。両方新規なら「タグ確認 → 板名確認」の順になる -->
    <ConfirmUnknownMiBoardDialog :unknown_mi_boards="unknown_mi_boards" :is_requested_submit="is_requested_submit"
        @requested_confirm="confirm_mi_board_submit()" @requested_cancel="cancel_mi_board_submit()"
        ref="confirm_unknown_mi_board_dialog" />
    <ConfirmCloseKFTLTabDialog :tab_label="pending_close_tab_label"
        @requested_confirm="confirm_close_tab()" @requested_cancel="cancel_close_tab()"
        ref="confirm_close_kftl_tab_dialog" />
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'

import type { KFTLProps } from './kftl-props'
import type { KFTLViewEmits } from './kftl-view-emits'

import KFTLLineLabel from './kftl-line-label.vue'
import KFTLTemplateDialog from '../dialogs/kftl-template-dialog.vue'
import ConfirmUnknownMiBoardDialog from '../dialogs/confirm-unknown-mi-board-dialog.vue'
import ConfirmCloseKFTLTabDialog from '../dialogs/confirm-close-kftl-tab-dialog.vue'
import { useKftlView } from '@/classes/use-kftl-view'

const props = defineProps<KFTLProps>()
const emits = defineEmits<KFTLViewEmits>()

const {
    // Template refs
    kftl_template_dialog,
    confirm_unknown_mi_board_dialog,
    confirm_close_kftl_tab_dialog,
    kftl_text_area,
    kftl_line_label_wrap,

    // Confirm unknown mi board
    unknown_mi_boards,
    cancel_mi_board_submit,
    confirm_mi_board_submit,

    // Tabs
    tabs,
    active_tab_id_model,
    is_tab_locked,
    tab_bar_height,
    tab_label,
    add_tab,
    request_close_tab,
    confirm_close_tab,
    cancel_close_tab,
    pending_close_tab_label,

    // State
    text_area_content,
    text_area_element_id,
    line_label_datas,
    line_label_styles,
    is_requested_submit,
    show_confirm_unknown_tag_dialog,
    unknown_tags,

    // Dialog UI
    confirm_dialog_ui,

    // Computed
    text_area_width_px,
    text_area_height_px,
    line_label_width_px,
    line_label_height_px,
    kftl_input_height_px,
    kftl_input_width_px,
    title_height_px,

    // Business logic
    submit,
    cancel_submit,
    confirm_submit,
    show_kftl_template_dialog,
    paste_template,
    focus_kftl_text_area,
    onTextAreaInput,
    update_line_labels,

    // Event relay objects
    errorMessageRelayHandlers,
} = useKftlView({ props, emits })

defineExpose({ focus_kftl_text_area })
</script>

<style lang="css" scoped>
.kftl_text_area_wrap {
    height: 100%;
    width: calc(v-bind(text_area_width_px));
}

.kftl_text_area {
    height: calc(v-bind(text_area_height_px));
    width: calc(v-bind(text_area_width_px));
    resize: none;
    font-size: 1em;
    line-height: 24px;
}

.line_label_wrap {
    color: rgb(var(--v-theme-background-focused));
    padding-right: 16px;
    height: calc(v-bind(line_label_height_px));
    width: calc(v-bind(line_label_width_px));
    text-align: right;
}

textarea {
    border: solid 1px silver;
}

.kftl_input {
    height: calc(v-bind(kftl_input_height_px));
    width: calc(v-bind(kftl_input_width_px));
    overflow-y: scroll;
}

.kftl_line_label {
    overflow-y: hidden;
}

.kftl_line_label::-webkit-scrollbar {
    display: none;
}

/* title_height はテキストエリアの高さ計算に使う定数。
   v-card-title に height prop は無いので、ここで実寸を定数へ固定しておく。
   既定の縦パディングと line-height を潰さないと、タブ列より背が高くなって
   タイトルとテキストエリアのあいだに余白ができる */
.kftl_title {
    height: calc(v-bind(title_height_px));
    min-height: calc(v-bind(title_height_px));
    padding: 0 8px;
    line-height: normal;
    display: flex;
    align-items: center;
    overflow: hidden;
}

.kftl_title > .v-row {
    flex: 1 1 auto;
    min-width: 0;
}

/* タブ列は残り幅を全部使う。テンプレート/保存ボタンは右端に固定 */
.kftl_tab_col {
    min-width: 0;
    overflow: hidden;
}

/* v-tabs の既定は大文字化。半角英数のタブ名が化けるので戻す */
.kftl_tab {
    min-width: 88px;
    max-width: 180px;
    text-transform: none;
}

.kftl_tab_label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.kftl_tab_close {
    cursor: pointer;
}
</style>
