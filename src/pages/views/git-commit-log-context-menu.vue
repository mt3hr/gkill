<template>
    <AddTagDialog />
    <AddTextDialog />
    <ConfirmReKyouDialog />
</template>
<script lang="ts" setup>
import type { GitCommitLogContextMenuProps } from './git-commit-log-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import AddTagDialog from '../dialogs/add-tag-dialog.vue';
import AddTextDialog from '../dialogs/add-text-dialog.vue';
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue';

const props = defineProps<GitCommitLogContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
</script>
