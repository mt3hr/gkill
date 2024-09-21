<template>
    <AddTagDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
        :kyou="cloned_kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <AddTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <ConfirmReKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script lang="ts" setup>
import type { GitCommitLogContextMenuProps } from './git-commit-log-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import AddTagDialog from '../dialogs/add-tag-dialog.vue';
import AddTextDialog from '../dialogs/add-text-dialog.vue';
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue';
import type { Kyou } from '@/classes/datas/kyou';
import { type Ref, ref } from 'vue';

const props = defineProps<GitCommitLogContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>
