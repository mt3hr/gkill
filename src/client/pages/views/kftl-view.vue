<template>
    <v-card class="kftl_view">
        <v-card-title :height="title_height">
            <v-row>
                <v-col cols="auto">
                    {{ i18n.global.t("KFTL_ADD_KYOU_TITLE") }}
                </v-col>
                <v-spacer />
                <v-col cols="auto">
                    <v-btn dark color="primary" @click="show_kftl_template_dialog" :disabled="is_requested_submit">{{
                        i18n.global.t("KFTL_TEMPLATE_TITLE") }}</v-btn>
                </v-col>
                <v-col cols="auto">
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
                        <div class="kftl_line_label line_label_wrap">
                            <KFTLLineLabel v-for="(line_label_data, index) in line_label_datas"
                                :key="index" :application_config="application_config"
                                :gkill_api="gkill_api" :line_label_data="line_label_data"
                                :style="line_label_styles[index]" />
                        </div>
                    </td>
                    <td>
                        <div class="kftl_text_area_wrap">
                            <textarea id="kftl_text_area" class="kftl_text_area" v-model="text_area_content"
                                :readonly="is_requested_submit" autofocus></textarea>
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
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'

import type { KFTLProps } from './kftl-props'
import type { KFTLViewEmits } from './kftl-view-emits'

import KFTLLineLabel from './kftl-line-label.vue'
import KFTLTemplateDialog from '../dialogs/kftl-template-dialog.vue'
import ConfirmUnknownMiBoardDialog from '../dialogs/confirm-unknown-mi-board-dialog.vue'
import { useKftlView } from '@/classes/use-kftl-view'

const props = defineProps<KFTLProps>()
const emits = defineEmits<KFTLViewEmits>()

const {
    // Template refs
    kftl_template_dialog,
    confirm_unknown_mi_board_dialog,

    // Confirm unknown mi board
    unknown_mi_boards,
    cancel_mi_board_submit,
    confirm_mi_board_submit,

    // State
    text_area_content,
    line_label_datas,
    line_label_styles,
    is_requested_submit,
    title_height,
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

    // Business logic
    submit,
    cancel_submit,
    confirm_submit,
    show_kftl_template_dialog,
    paste_template,
    focus_kftl_text_area,

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
</style>
