<template>
    <!-- タグ版(add-tag-view.vue / kftl-view.vue)は同じマークアップを各Viewに手書きで複製しているが、
         板名は5経路(Mi追加/Mi編集/MiReKyou追加/MiReKyou編集/KFTL)から使うので共有部品にする -->
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
                                <span>{{ i18n.global.t("CONFIRM_UNKNOWN_MI_BOARD_TITLE") }}</span>
                            </v-col>
                        </v-row>
                    </v-card-title>
                    <v-card-text>
                        {{ i18n.global.t("CONFIRM_UNKNOWN_MI_BOARD_MESSAGE") }}
                        <v-list density="compact">
                            <v-list-item v-for="board_name in unknown_mi_boards" :key="board_name">
                                <v-list-item-title>{{ board_name }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-card-text>
                    <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
                        <v-col cols="auto" class="pa-0 ma-0">
                            <v-btn dark color="primary" @click="confirm()" :disabled="is_requested_submit">{{
                                i18n.global.t("SAVE_TITLE") }}</v-btn>
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
import type { ConfirmUnknownMiBoardDialogEmits } from './confirm-unknown-mi-board-dialog-emits'
import type { ConfirmUnknownMiBoardDialogProps } from './confirm-unknown-mi-board-dialog-props'
import { useConfirmUnknownMiBoardDialog } from '@/classes/use-confirm-unknown-mi-board-dialog'

const props = defineProps<ConfirmUnknownMiBoardDialogProps>()
const emits = defineEmits<ConfirmUnknownMiBoardDialogEmits>()
const { is_show_dialog, ui, show, hide, confirm, cancel } = useConfirmUnknownMiBoardDialog({ props, emits })
defineExpose({ show, hide })
</script>
