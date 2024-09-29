<template>
    <AttachedTextContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_text(text)]" :kyou="kyou" :last_added_tag="last_added_tag"
        :text="text" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <ConfirmDeleteTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_text(text)]" :kyou="kyou" :last_added_tag="last_added_tag"
        :text="text" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <EditTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_text(text)]" :kyou="kyou" :last_added_tag="last_added_tag"
        :text="text" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { AttachedTextProps } from './attached-text-props'
import type { KyouViewEmits } from './kyou-view-emits'
import AttachedTextContextMenu from './attached-text-context-menu.vue'
import ConfirmDeleteTextDialog from '../dialogs/confirm-delete-text-dialog.vue'
import EditTextDialog from '../dialogs/edit-text-dialog.vue'
import { Text } from '@/classes/datas/text'
import { Kyou } from '@/classes/datas/kyou'
import { InfoIdentifier } from '@/classes/datas/info-identifier'

const props = defineProps<AttachedTextProps>()
const emits = defineEmits<KyouViewEmits>()

function generate_info_identifer_from_text(text: Text): InfoIdentifier {
    const info_identifer = new InfoIdentifier()
    info_identifer.create_time = text.create_time
    info_identifer.id = text.id
    info_identifer.update_time = text.update_time
    return info_identifer
}
</script>
