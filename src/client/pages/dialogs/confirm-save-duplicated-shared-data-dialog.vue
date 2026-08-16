<template>
    <!-- Android のタスク復帰で同じ共有インテントが再配送されると、初回とまったく同じ内容が届く。
         再配送か意図的な再共有かは内容から見分けられないので、捨てずにここで選ばせる -->
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
                                <span>{{ i18n.global.t("CONFIRM_SAVE_DUPLICATED_SHARED_DATA_TITLE") }}</span>
                            </v-col>
                        </v-row>
                    </v-card-title>
                    <v-card-text>
                        {{ i18n.global.t("CONFIRM_SAVE_DUPLICATED_SHARED_DATA_MESSAGE") }}
                        <v-list density="compact" v-if="entry">
                            <v-list-item>
                                <v-list-item-title class="shared_data_summary">{{ shared_summary }}</v-list-item-title>
                                <v-list-item-subtitle>{{ format_time(new Date(entry.saved_at)) }}</v-list-item-subtitle>
                            </v-list-item>
                        </v-list>
                    </v-card-text>
                    <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
                        <v-col cols="auto" class="pa-0 ma-0">
                            <v-btn dark color="primary" @click="confirm()">{{
                                i18n.global.t("SAVE_ANYWAY_TITLE") }}</v-btn>
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
import { computed, type Ref, ref } from 'vue'
import { i18n } from '@/i18n'
import type { ConfirmSaveDuplicatedSharedDataDialogEmits } from './confirm-save-duplicated-shared-data-dialog-emits'
import type { ConfirmSaveDuplicatedSharedDataDialogProps } from './confirm-save-duplicated-shared-data-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { format_time } from '@/classes/format-date-time'

const props = defineProps<ConfirmSaveDuplicatedSharedDataDialogProps>()
const emits = defineEmits<ConfirmSaveDuplicatedSharedDataDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
const ui = useFloatingDialog("confirm-save-duplicated-shared-data-dialog", {
    centerMode: "always",
    onEscape: () => cancel(),
})

// 何が重複したのかを出す。URL 共有が大半なので URL を優先する
const shared_summary = computed(() => {
    const payload = props.entry?.payload
    if (!payload) {
        return ""
    }
    return payload.url || payload.text || payload.title
})

async function show(): Promise<void> {
    is_show_dialog.value = true
}

async function hide(): Promise<void> {
    close_dialog_via_history(is_show_dialog)
}

function confirm(): void {
    hide()
    emits('requested_save')
}

function cancel(): void {
    hide()
    emits('requested_cancel')
}
</script>
<style lang="css" scoped>
.shared_data_summary {
    white-space: pre-wrap;
    word-break: break-all;
}
</style>
