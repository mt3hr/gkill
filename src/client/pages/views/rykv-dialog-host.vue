<template>
  <RykvDialogHostItem
    v-for="item in dialogs"
    :key="item.id"
    :item="item"
    :application_config="application_config"
    :gkill_api="gkill_api"
    :enable_context_menu="enable_context_menu"
    :enable_dialog="enable_dialog"
    v-on="crudRelayHandlers"
  />
</template>

<script setup lang="ts">
import type { OpenedRykvDialog } from './rykv-dialog-kind'
import type { GkillPropsBase } from './gkill-props-base'
import type { KyouViewEmits } from './kyou-view-emits'
import RykvDialogHostItem from './rykv-dialog-host-item.vue'
import { useRykvDialogHost } from '@/classes/use-rykv-dialog-host'

interface RykvDialogHostProps extends GkillPropsBase {
  dialogs: Array<OpenedRykvDialog>
  enable_context_menu: boolean
  enable_dialog: boolean
}

defineProps<RykvDialogHostProps>()

interface RykvDialogHostEmits extends KyouViewEmits {
  (e: 'closed', id: string): void
}

const emits = defineEmits<RykvDialogHostEmits>()

const {
    crudRelayHandlers,
} = useRykvDialogHost({ emits })
</script>
