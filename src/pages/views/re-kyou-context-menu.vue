<template>
    <EditReKyouDialog />
    <AddTagDialog />
    <AddTextDialog />
    <ConfirmReKyouDialog />
    <ConfirmDeleteKyouDialog />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits';
import type { ReKyouContextMenuProps } from './re-kyou-context-menu-props';
import EditReKyouDialog from '../dialogs/edit-re-kyou-dialog.vue';
import AddTagDialog from '../dialogs/add-tag-dialog.vue';
import AddTextDialog from '../dialogs/add-text-dialog.vue';
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';

const props = defineProps<ReKyouContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
</script>