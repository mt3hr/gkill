<template>
    <EditUrLogDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag" :urlog="kyou.typed_urlog"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <AddTagDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
        :kyou="kyou" :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <AddTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <ConfirmDeleteKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="latest_kyou_identifier" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(message) => emits('received_messages', message)" />
    <ConfirmReKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="latest_kyou_identifier" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits'
import type { URLogContextMenuProps } from './ur-log-context-menu-props'
import EditUrLogDialog from '../dialogs/edit-ur-log-dialog.vue'
import AddTagDialog from '../dialogs/add-tag-dialog.vue'
import AddTextDialog from '../dialogs/add-text-dialog.vue'
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue'
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue'
import { InfoIdentifier } from '@/classes/datas/info-identifier'
import { computed, type ComputedRef } from 'vue'

const props = defineProps<URLogContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
const latest_kyou_identifier: ComputedRef<Array<InfoIdentifier>> = computed(() => {
    const identifier = new InfoIdentifier()
    identifier.create_time = props.kyou.create_time
    identifier.id = props.kyou.id
    identifier.update_time = props.kyou.update_time
    return [identifier]
})
</script>
