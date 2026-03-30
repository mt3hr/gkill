<template>
    <Teleport to="body" v-if="is_show_dialog" >
        <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

        <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
            :class="ui.isTransparent.value ? 'is-transparent' : ''">
            <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
                @touchstart="ui.onHeaderPointerDown">
                <div class="gkill-floating-dialog__title"></div>
                <div class="gkill-floating-dialog__spacer"></div>
                <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
                    :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat"> 
          <v-icon>mdi-close</v-icon>
        </v-btn>
            </div>

            <div class="gkill-floating-dialog__body"> 

                <v-card v-if="is_show_dialog" class="kyou_list_view_dialog_view pa-2" style="flex: 1; min-height: 0; width: 100%;">
                    <KyouListView :kyou_height="180" :width="400" :list_height="list_height"
                        :application_config="application_config" :gkill_api="gkill_api" :matched_kyous="model_value!"
                        :query="new FindKyouQuery()" :is_focused_list="true"
                        :closable="false" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                        :is_readonly_mi_check="true" :show_checkbox="true" :show_footer="false"
                        :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true" :show_content_only="false"
                        :show_timeis_plaing_end_button="false" :show_rep_name="show_rep_name"
                        :force_show_latest_kyou_info="force_show_latest_kyou_info"
                        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
                        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
                        @focused_kyou="(kyou: Kyou) => emits('focused_kyou', kyou)"
                        @clicked_kyou="(kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) }"
                        @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
                        @requested_open_rykv_dialog="(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload)"
                        ref="kyou_list_views"
                        @deleted_kyou="(deleted_kyou: Kyou) => onDeletedKyou(deleted_kyou)"
                        @deleted_tag="(_deleted_tag: Tag) => { /* intentionally ignored */ }" @deleted_text="(_deleted_text: Text) => { /* intentionally ignored */ }"
                        @deleted_notification="(_deleted_notification: Notification) => { /* intentionally ignored */ }"
                        @registered_kyou="(_registered_kyou: Kyou) => { /* intentionally ignored */ }"
                        @registered_tag="(_registered_tag: Tag) => { /* intentionally ignored */ }"
                        @registered_text="(_registered_text: Text) => { /* intentionally ignored */ }"
                        @registered_notification="(_registered_notification: Notification) => { /* intentionally ignored */ }"
                        @updated_kyou="(updated_kyou: Kyou) => reload_kyou(updated_kyou)"
                        @updated_tag="(_updated_tag: Tag) => { /* intentionally ignored */ }" @updated_text="(_updated_text: Text) => { /* intentionally ignored */ }"
                        @updated_notification="(_updated_notification: Notification) => { /* intentionally ignored */ }" />
                </v-card>
                <v-card variant="text" :ripple="false" :link="false" class="px-2" style="flex-shrink: 0;">
                    <v-row no-gutters>
                        <v-col v-if="model_value && model_value.length" cols="auto" class="py-3">
                            {{ model_value.length }}{{ i18n.global.t("N_COUNT_ITEMS_TITLE") }}
                        </v-col>
                        <v-spacer />
                    </v-row>
                </v-card>
            </div>
        </div>
    </Teleport>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import KyouListView from '../views/kyou-list-view.vue';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import type { Kyou } from '@/classes/datas/kyou';
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { KyouListViewDialogProps } from './kyou-list-view-dialog-props';
import type { KyouListViewEmits } from '../views/kyou-list-view-emits';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import type { RykvDialogKind, RykvDialogPayload } from '../views/rykv-dialog-kind';

defineProps<KyouListViewDialogProps>()
const model_value = defineModel<Array<Kyou>>()
const emits = defineEmits<KyouListViewEmits>()

defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("kyou-list-view-dialog", {
  centerMode: "always",
  onEscape: () => hide(),
})

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    (async (): Promise<void> => {
        const kyous_list = model_value.value!
        for (let j = 0; j < kyous_list.length; j++) {
            const kyou_in_list = kyous_list[j]
            if (kyou.id === kyou_in_list.id) {
                const updated_kyou = kyou.clone()
                await updated_kyou.reload(false, true)
                await updated_kyou.load_all()
                kyous_list.splice(j, 1, updated_kyou)
            }
        }
    })();
}

function onDeletedKyou(deletedKyou: Kyou): void {
    if (!model_value.value) {
        return
    }
    for (let i = model_value.value.length - 1; i >= 0; i--) {
        if (model_value.value[i].id === deletedKyou.id) {
            model_value.value.splice(i, 1)
        }
    }
    emits('deleted_kyou', deletedKyou)
}
</script>

<style scoped lang="css">
.kyou_list_view_dialog_view,
.kyou_list_view_dialog {
    overflow-y: hidden !important;
}

.kyou_detail_view,
.kyou_list_view,
.v-dialog .v-card {
    overflow-y: hidden !important;
}

.kyou_list_view_dialog_view :deep(.kyou_list_view) {
    width: 100% !important;
}
</style>

