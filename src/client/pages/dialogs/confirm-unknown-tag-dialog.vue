<template>
    <!-- 板名版 confirm-unknown-mi-board-dialog.vue と対の共有部品。
         add-tag-view / kftl-view / Kyouの追加・編集18画面から使う -->
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
                                <span>{{ i18n.global.t("CONFIRM_UNKNOWN_TAG_TITLE") }}</span>
                            </v-col>
                        </v-row>
                    </v-card-title>
                    <v-card-text>
                        {{ i18n.global.t("CONFIRM_UNKNOWN_TAG_MESSAGE") }}
                        <v-list density="compact">
                            <v-list-item v-for="tag_name in unknown_tags" :key="tag_name">
                                <v-list-item-title>{{ tag_name }}</v-list-item-title>
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
import type { ConfirmUnknownTagDialogEmits } from './confirm-unknown-tag-dialog-emits'
import type { ConfirmUnknownTagDialogProps } from './confirm-unknown-tag-dialog-props'
import { useConfirmUnknownTagDialog } from '@/classes/use-confirm-unknown-tag-dialog'

const props = defineProps<ConfirmUnknownTagDialogProps>()
const emits = defineEmits<ConfirmUnknownTagDialogEmits>()
const { is_show_dialog, ui, show, hide, confirm, cancel } = useConfirmUnknownTagDialog({ props, emits })
defineExpose({ show, hide })
</script>
