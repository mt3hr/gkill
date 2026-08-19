<template>
  <Teleport to="body" v-if="is_show_dialog" >
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
  <v-checkbox v-model="ui.isTransparent.value" color="white"    size="small" variant="flat" 
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat"> 
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card variant="flat" class="pa-2">

        <ShareKyousListLinkView v-if="share_kyou_list_info" :application_config="application_config"
          :gkill_api="gkill_api" :share_kyou_list_info="share_kyou_list_info"
          @updated_share_kyou_list_info="(share_kyou_list_info: ShareKyousInfo) => emits('updated_share_kyou_list_info', share_kyou_list_info)"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_close_dialog="hide()" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { ShareKyousListLinkDialogEmits } from './share-kyou-list-link-dialog-emits'
import type { ShareKyousListLinkDialogProps } from './share-kyou-list-link-dialog-props'
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import ShareKyousListLinkView from '../views/share-kyou-link-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useShareKyouListLinkDialog } from '@/classes/use-share-kyou-list-link-dialog'

const props = defineProps<ShareKyousListLinkDialogProps>()
const emits = defineEmits<ShareKyousListLinkDialogEmits>()
const { is_show_dialog, ui, share_kyou_list_info, show, hide } = useShareKyouListLinkDialog({ props, emits })
defineExpose({ show, hide })
</script>

