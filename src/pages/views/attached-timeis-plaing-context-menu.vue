<template>
    <EditTimeIsDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_timeis(cloned_timeis)]" :kyou="cloned_kyou"
        :last_added_tag="last_added_tag" :timeis="cloned_timeis"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <KyouHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_timeis(cloned_timeis)]" :kyou="cloned_kyou"
        :last_added_tag="last_added_tag" :timeis="cloned_timeis"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <ConfirmDeleteKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_timeis(cloned_timeis)]" :kyou="cloned_kyou"
        :last_added_tag="last_added_tag" :timeis="cloned_timeis"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { AttachedTimeisPlaingContextMenuProps } from './attached-timeis-plaing-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { TimeIs } from '@/classes/datas/time-is';
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue';
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';
import { InfoIdentifier } from '@/classes/datas/info-identifier';
import type { Kyou } from '@/classes/datas/kyou';

const props = defineProps<AttachedTimeisPlaingContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_timeis: Ref<TimeIs> = ref(await props.timeis.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());

function generate_info_identifer_from_timeis(timeis: TimeIs): InfoIdentifier {
    const info_identifer = new InfoIdentifier()
    info_identifer.create_time = timeis.create_time
    info_identifer.id = timeis.id
    info_identifer.update_time = timeis.update_time
    return info_identifer
}
</script>
