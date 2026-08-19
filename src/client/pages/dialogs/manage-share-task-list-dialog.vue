<template>
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
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>

            <div class="gkill-floating-dialog__body">
              <v-card variant="flat" class="pa-2">
               <ManageShareKyousListView :application_config="application_config" :gkill_api="gkill_api"
                    :share_kyou_list_infos="share_kyou_list_infos"
                    @requested_show_confirm_delete_share_kyou_list_dialog="show_confirm_delete_share_kyou_list_dialog"
                    @requested_show_share_kyou_link_dialog="show_share_kyou_list_link_dialog" />
                <ShareKyousListLinkDialog :share_kyou_list_info="share_kyou_list"
                    :application_config="application_config" :gkill_api="gkill_api"
                    @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
                    @updated_share_kyou_list_info="reload_share_kyou_list_infos()"
                    @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
                    ref="share_kyou_list_link_dialog" />
                <ConfirmDeleteShareKyousListDialog :share_kyou_list_info="share_kyou_link"
                    :application_config="application_config" :gkill_api="gkill_api"
                    @requested_delete_share_kyou_link_info="(share_kyou_list_infos: ShareKyousInfo) => delete_share_kyou_link_info(share_kyou_list_infos)"
                    @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
                    @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
                    ref="confirm_delete_share_kyou_list_dialog" />
              </v-card>
</div>
        </div>
    </Teleport>
</template>
<script lang="ts" setup>
import type { ManageShareKyousLinkDialogEmits } from './manage-share-task-link-dialog-emits'
import type { ManageShareKyousLinkDialogProps } from './manage-share-task-link-dialog-props'
import ManageShareKyousListView from '../views/manage-share-task-list-view.vue'
import ShareKyousListLinkDialog from './share-kyou-list-link-dialog.vue'
import ConfirmDeleteShareKyousListDialog from './confirm-delete-share-kyou-list-dialog.vue'
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useManageShareTaskListDialog } from '@/classes/use-manage-share-task-list-dialog'

const props = defineProps<ManageShareKyousLinkDialogProps>()
const emits = defineEmits<ManageShareKyousLinkDialogEmits>()
const { share_kyou_list_link_dialog, confirm_delete_share_kyou_list_dialog, share_kyou_list_infos, share_kyou_list, share_kyou_link, is_show_dialog, ui, show, hide, show_confirm_delete_share_kyou_list_dialog, show_share_kyou_list_link_dialog, delete_share_kyou_link_info, reload_share_kyou_list_infos } = useManageShareTaskListDialog({ props, emits })
defineExpose({ show, hide })
</script>

