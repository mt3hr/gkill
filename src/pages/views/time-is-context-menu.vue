<template>
    <EditTimeIsDialog />
    <AddTagDialog />
    <AddTextDialog />
    <ConfirmReKyouDialog />
    <ConfirmDeleteKyouDialog />
</template>
<script lang="ts" setup>
import type { TimeIsContextMenuProps } from './time-is-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue';
import AddTagDialog from '../dialogs/add-tag-dialog.vue';
import AddTextDialog from '../dialogs/add-text-dialog.vue';
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';

const props = defineProps<TimeIsContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
</script>
