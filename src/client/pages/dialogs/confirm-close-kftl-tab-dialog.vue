<template>
    <!-- 内容が残っているタブを × したときの確認。空のタブは呼び出し元が確認なしで閉じる -->
    <Teleport to="body" v-if="is_show_dialog">
        <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />
        <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
            :class="ui.isTransparent.value ? 'is-transparent' : ''">
            <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
                @touchstart="ui.onHeaderPointerDown">
                <div class="gkill-floating-dialog__title"></div>
                <div class="gkill-floating-dialog__spacer"></div>
                <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
                    :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="cancel()" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>
            <div class="gkill-floating-dialog__body">
                <v-card variant="flat" class="pa-2">
                    <v-card-title>
                        <v-row class="pa-0 ma-0">
                            <v-col cols="auto" class="pa-0 ma-0">
                                <span>{{ i18n.global.t("CONFIRM_CLOSE_KFTL_TAB_TITLE") }}</span>
                            </v-col>
                        </v-row>
                    </v-card-title>
                    <v-card-text>
                        {{ i18n.global.t("CONFIRM_CLOSE_KFTL_TAB_MESSAGE") }}
                        <v-list density="compact" v-if="tab_label !== ''">
                            <v-list-item>
                                <v-list-item-title>{{ tab_label }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-card-text>
                    <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
                        <v-col cols="auto" class="pa-0 ma-0">
                            <v-btn dark color="primary" @click="confirm()">{{
                                i18n.global.t("KFTL_TAB_CLOSE_TITLE") }}</v-btn>
                        </v-col>
                        <v-col cols="auto" class="pa-0 ma-0">
                            <v-btn @click="cancel()">{{ i18n.global.t("CANCEL_TITLE") }}</v-btn>
                        </v-col>
                        <v-spacer />
                    </v-row>
                </v-card>
            </div>
        </div>
    </Teleport>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { ConfirmCloseKFTLTabDialogEmits } from './confirm-close-kftl-tab-dialog-emits'
import type { ConfirmCloseKFTLTabDialogProps } from './confirm-close-kftl-tab-dialog-props'
import { useConfirmCloseKFTLTabDialog } from '@/classes/use-confirm-close-kftl-tab-dialog'

const props = defineProps<ConfirmCloseKFTLTabDialogProps>()
const emits = defineEmits<ConfirmCloseKFTLTabDialogEmits>()
const { is_show_dialog, ui, show, hide, confirm, cancel } = useConfirmCloseKFTLTabDialog({ props, emits })
defineExpose({ show, hide })
</script>
