<template>
  <KyouDialog
    v-if="item.kind === 'kyou'"
    ref="dialog"
    :application_config="application_config"
    :gkill_api="gkill_api"
    :highlight_targets="[]"
    :kyou="item.kyou"
    :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog"
    :is_readonly_mi_check="false"
    :show_timeis_plaing_end_button="false"
    v-on="dialog_events"
  />
  <EditKmemoDialog v-else-if="item.kind === 'edit_kmemo'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditKCDialog v-else-if="item.kind === 'edit_kc'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditMiDialog v-else-if="item.kind === 'edit_mi'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditNlogDialog v-else-if="item.kind === 'edit_nlog'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditLantanaDialog v-else-if="item.kind === 'edit_lantana'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditTimeIsDialog v-else-if="item.kind === 'edit_timeis'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditURLogDialog v-else-if="item.kind === 'edit_urlog'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditIDFKyouDialog v-else-if="item.kind === 'edit_idf_kyou'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditReKyouDialog v-else-if="item.kind === 'edit_re_kyou'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <AddTagDialog v-else-if="item.kind === 'add_tag'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <AddTextDialog v-else-if="item.kind === 'add_text'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <AddNotificationDialog v-else-if="item.kind === 'add_notification'" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
    v-on="dialog_events" />
  <ConfirmDeleteIDFKyouDialog v-else-if="item.kind === 'confirm_delete_kyou'" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
    v-on="dialog_events" />
  <ConfirmReKyouDialog v-else-if="item.kind === 'confirm_re_kyou'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <KyouHistoriesDialog v-else-if="item.kind === 'kyou_histories'" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditTagDialog v-else-if="item.kind === 'edit_tag' && payload_tag" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :tag="payload_tag"
    :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
    v-on="dialog_events" />
  <ConfirmDeleteTagDialog v-else-if="item.kind === 'confirm_delete_tag' && payload_tag" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :tag="payload_tag" :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog" v-on="dialog_events" />
  <TagHistoriesDialog v-else-if="item.kind === 'tag_histories' && payload_tag" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :tag="payload_tag" :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditTextDialog v-else-if="item.kind === 'edit_text' && payload_text" ref="dialog" :application_config="application_config"
    :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou" :text="payload_text"
    :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
    v-on="dialog_events" />
  <ConfirmDeleteTextDialog v-else-if="item.kind === 'confirm_delete_text' && payload_text" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :text="payload_text" :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog" v-on="dialog_events" />
  <TextHistoriesDialog v-else-if="item.kind === 'text_histories' && payload_text" ref="dialog"
    :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]" :kyou="item.kyou"
    :text="payload_text" :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog" v-on="dialog_events" />
  <EditNotificationDialog
    v-else-if="item.kind === 'edit_notification' && payload_notification"
    ref="dialog"
    :application_config="application_config"
    :gkill_api="gkill_api"
    :highlight_targets="[]"
    :kyou="item.kyou"
    :notification="payload_notification"
    :last_added_tag="last_added_tag"
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
    :last_added_tag="last_added_tag"
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
    :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog"
    v-on="dialog_events"
  />
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
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
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Notification } from '@/classes/datas/notification'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Kyou } from '@/classes/datas/kyou'
import type { OpenedRykvDialog } from './rykv-dialog-kind'
import type { GkillPropsBase } from './gkill-props-base'
import type { KyouDialogEmits } from './kyou-dialog-emits'

interface RykvDialogHostItemProps extends GkillPropsBase {
  item: OpenedRykvDialog
  last_added_tag: string
  enable_context_menu: boolean
  enable_dialog: boolean
}

const props = defineProps<RykvDialogHostItemProps>()

interface RykvDialogHostItemEmits extends KyouDialogEmits {
  (e: 'closed', id: string): void
}

const emits = defineEmits<RykvDialogHostItemEmits>()

const dialog = ref<any>(null)
const payload_tag = computed(() => (props.item.payload ?? null) as Tag | null)
const payload_text = computed(() => (props.item.payload ?? null) as Text | null)
const payload_notification = computed(() => (props.item.payload ?? null) as Notification | null)

const dialog_events = {
  closed: () => emits('closed', props.item.id),
  deleted_kyou: (kyou: Kyou) => emits('deleted_kyou', kyou),
  deleted_tag: (tag: Tag) => emits('deleted_tag', tag),
  deleted_text: (text: Text) => emits('deleted_text', text),
  deleted_notification: (notification: Notification) => emits('deleted_notification', notification),
  registered_kyou: (kyou: Kyou) => emits('registered_kyou', kyou),
  registered_tag: (tag: Tag) => emits('registered_tag', tag),
  registered_text: (text: Text) => emits('registered_text', text),
  registered_notification: (notification: Notification) => emits('registered_notification', notification),
  updated_kyou: (kyou: Kyou) => emits('updated_kyou', kyou),
  updated_tag: (tag: Tag) => emits('updated_tag', tag),
  updated_text: (text: Text) => emits('updated_text', text),
  updated_notification: (notification: Notification) => emits('updated_notification', notification),
  received_errors: (errors: Array<GkillError>) => emits('received_errors', errors),
  received_messages: (messages: Array<GkillMessage>) => emits('received_messages', messages),
  requested_reload_kyou: (kyou: Kyou) => emits('requested_reload_kyou', kyou),
  requested_reload_list: () => emits('requested_reload_list'),
  requested_update_check_kyous: (kyous: Array<Kyou>, is_checked: boolean) =>
    emits('requested_update_check_kyous', kyous, is_checked),
}

onMounted(async () => {
  await nextTick()
  dialog.value?.show?.()
})
</script>
