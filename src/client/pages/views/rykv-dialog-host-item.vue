<template>
  <KyouDialog
    v-if="item.kind === 'kyou'"
    ref="dialog"
    :application_config="application_config"
    :gkill_api="gkill_api"
    :highlight_targets="[]"
    :kyou="item.kyou"
    :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog"
    :is_readonly_mi_check="false"
    :show_timeis_plaing_end_button="false"
    v-on="dialog_events"
  />
  <EditKmemoDialog v-else-if="item.kind === 'edit_kmemo'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditKCDialog v-else-if="item.kind === 'edit_kc'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditMiDialog v-else-if="item.kind === 'edit_mi'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditNlogDialog v-else-if="item.kind === 'edit_nlog'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditLantanaDialog v-else-if="item.kind === 'edit_lantana'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditTimeIsDialog v-else-if="item.kind === 'edit_timeis'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditURLogDialog v-else-if="item.kind === 'edit_urlog'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditIDFKyouDialog v-else-if="item.kind === 'edit_idf_kyou'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditReKyouDialog v-else-if="item.kind === 'edit_re_kyou'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <AddTagDialog v-else-if="item.kind === 'add_tag'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <AddTextDialog v-else-if="item.kind === 'add_text'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <AddNotificationDialog v-else-if="item.kind === 'add_notification'" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
    v-on="dialog_events" />
  <ConfirmDeleteIDFKyouDialog v-else-if="item.kind === 'confirm_delete_kyou'" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
    v-on="dialog_events" />
  <ConfirmReKyouDialog v-else-if="item.kind === 'confirm_re_kyou'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <KyouHistoriesDialog v-else-if="item.kind === 'kyou_histories'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditTagDialog v-else-if="item.kind === 'edit_tag' && payload_tag" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :tag="payload_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
    v-on="dialog_events" />
  <ConfirmDeleteTagDialog v-else-if="item.kind === 'confirm_delete_tag' && payload_tag" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :tag="payload_tag" :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog" v-on="dialog_events" />
  <TagHistoriesDialog v-else-if="item.kind === 'tag_histories' && payload_tag" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :tag="payload_tag" :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditTextDialog v-else-if="item.kind === 'edit_text' && payload_text" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :text="payload_text"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
    v-on="dialog_events" />
  <ConfirmDeleteTextDialog v-else-if="item.kind === 'confirm_delete_text' && payload_text" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :text="payload_text" :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog" v-on="dialog_events" />
  <TextHistoriesDialog v-else-if="item.kind === 'text_histories' && payload_text" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :text="payload_text" :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditNotificationDialog
    v-else-if="item.kind === 'edit_notification' && payload_notification"
    ref="dialog"
    :application_config="application_config"
    :gkill_api="gkill_api"
    :highlight_targets="[]"
    :kyou="item.kyou"
    :notification="payload_notification"
    :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog"
    v-on="dialog_events"
  />
  <ConfirmDeleteNotificationDialog
    v-else-if="item.kind === 'confirm_delete_notification' && payload_notification"
    ref="dialog"
    :application_config="application_config"
    :gkill_api="gkill_api"
    :highlight_targets="[]"
    :kyou="item.kyou"
    :notification="payload_notification"
    :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog"
    v-on="dialog_events"
  />
  <BrowseZipContentsDialog
    v-else-if="item.kind === 'browse_zip_contents'"
    ref="dialog"
    :application_config="application_config"
    :gkill_api="gkill_api"
    :highlight_targets="[]"
    :kyou="item.kyou"
    :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog"
    v-on="dialog_events"
  />
  <NotificationHistoriesDialog
    v-else-if="item.kind === 'notification_histories' && payload_notification"
    ref="dialog"
    :application_config="application_config"
    :gkill_api="gkill_api"
    :highlight_targets="[]"
    :kyou="item.kyou"
    :notification="payload_notification"
    :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog"
    v-on="dialog_events"
  />
</template>

<script setup lang="ts">
import BrowseZipContentsDialog from '../dialogs/browse-zip-contents-dialog.vue'
import AddNotificationDialog from '../dialogs/add-notification-dialog.vue'
import AddTagDialog from '../dialogs/add-tag-dialog.vue'
import AddTextDialog from '../dialogs/add-text-dialog.vue'
import ConfirmDeleteIDFKyouDialog from '../dialogs/confirm-delete-idf-kyou-dialog.vue'
import ConfirmDeleteNotificationDialog from '../dialogs/confirm-delete-notification-dialog.vue'
import ConfirmDeleteTagDialog from '../dialogs/confirm-delete-tag-dialog.vue'
import ConfirmDeleteTextDialog from '../dialogs/confirm-delete-text-dialog.vue'
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue'
import EditIDFKyouDialog from '../dialogs/edit-idf-kyou-dialog.vue'
import EditKCDialog from '../dialogs/edit-kc-dialog.vue'
import EditKmemoDialog from '../dialogs/edit-kmemo-dialog.vue'
import EditLantanaDialog from '../dialogs/edit-lantana-dialog.vue'
import EditMiDialog from '../dialogs/edit-mi-dialog.vue'
import EditNlogDialog from '../dialogs/edit-nlog-dialog.vue'
import EditNotificationDialog from '../dialogs/edit-notification-dialog.vue'
import EditReKyouDialog from '../dialogs/edit-re-kyou-dialog.vue'
import EditTagDialog from '../dialogs/edit-tag-dialog.vue'
import EditTextDialog from '../dialogs/edit-text-dialog.vue'
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue'
import EditURLogDialog from '../dialogs/edit-ur-log-dialog.vue'
import KyouDialog from '../dialogs/kyou-dialog.vue'
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue'
import NotificationHistoriesDialog from '../dialogs/notification-histories-dialog.vue'
import TagHistoriesDialog from '../dialogs/tag-histories-dialog.vue'
import TextHistoriesDialog from '../dialogs/text-histories-dialog.vue'
import type { OpenedRykvDialog } from './rykv-dialog-kind'
import type { GkillPropsBase } from './gkill-props-base'
import type { KyouDialogEmits } from './kyou-dialog-emits'
import { useRykvDialogHostItem } from '@/classes/use-rykv-dialog-host-item'

interface RykvDialogHostItemProps extends GkillPropsBase {
  item: OpenedRykvDialog
  enable_context_menu: boolean
  enable_dialog: boolean
}

const props = defineProps<RykvDialogHostItemProps>()

interface RykvDialogHostItemEmits extends KyouDialogEmits {
  (e: 'closed', id: string): void
}

const emits = defineEmits<RykvDialogHostItemEmits>()

const {
  dialog,
  payload_tag,
  payload_text,
  payload_notification,
  dialog_events,
} = useRykvDialogHostItem({ props, emits })

defineExpose({ dialog })
</script>
