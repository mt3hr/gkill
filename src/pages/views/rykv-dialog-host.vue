<template>
  <RykvDialogHostItem
    v-for="item in dialogs"
    :key="item.id"
    :item="item"
    :application_config="application_config"
    :gkill_api="gkill_api"
    :last_added_tag="last_added_tag"
    :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog"
    @closed="(...id: any[]) => emits('closed', id[0] as string)"
    @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
    @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
    @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
    @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
    @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
    @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
    @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
    @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
    @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
    @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
    @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
    @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
    @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
    @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
    @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
    @requested_reload_list="() => emits('requested_reload_list')"
    @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
    @requested_open_rykv_dialog="(...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2])"
  />
</template>

<script setup lang="ts">
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Notification } from '@/classes/datas/notification'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Kyou } from '@/classes/datas/kyou'
import type { OpenedRykvDialog } from './rykv-dialog-kind'
import type { GkillPropsBase } from './gkill-props-base'
import type { KyouViewEmits } from './kyou-view-emits'
import RykvDialogHostItem from './rykv-dialog-host-item.vue'

interface RykvDialogHostProps extends GkillPropsBase {
  dialogs: Array<OpenedRykvDialog>
  last_added_tag: string
  enable_context_menu: boolean
  enable_dialog: boolean
}

defineProps<RykvDialogHostProps>()

interface RykvDialogHostEmits extends KyouViewEmits {
  (e: 'closed', id: string): void
}

const emits = defineEmits<RykvDialogHostEmits>()
</script>
