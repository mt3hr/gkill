<template>
    <EditTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_text(cloned_text)]" :kyou="cloned_kyou"
        :last_added_tag="last_added_tag" :text="cloned_text"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <ConfirmDeleteTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_text(cloned_text)]" :kyou="cloned_kyou"
        :last_added_tag="last_added_tag" :text="cloned_text"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <TextHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_text(cloned_text)]" :kyou="cloned_kyou"
        :last_added_tag="last_added_tag" :text="cloned_text"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { AttachedTextContextMenuProps } from './attached-text-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { Text } from '@/classes/datas/text';
import EditTextDialog from '../dialogs/edit-text-dialog.vue';
import ConfirmDeleteTextDialog from '../dialogs/confirm-delete-text-dialog.vue';
import TextHistoriesDialog from '../dialogs/text-histories-dialog.vue';
import type { Kyou } from '@/classes/datas/kyou';
import { InfoIdentifier } from '@/classes/datas/info-identifier';

const props = defineProps<AttachedTextContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_text: Ref<Text> = ref(await props.text.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());

function generate_info_identifer_from_text(text: Text): InfoIdentifier {
    const info_identifer = new InfoIdentifier()
    info_identifer.create_time = text.create_time
    info_identifer.id = text.id
    info_identifer.update_time = text.update_time
    return info_identifer
}
</script>
