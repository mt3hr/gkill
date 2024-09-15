<template>
    <EditLantanaDialog />
    <AddTagDialog />
    <AddTextDialog />
    <ConfirmDeleteKyouDialog />
    <ConfirmReKyouDialog />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits';
import type { LantanaContextMenuProps } from './lantana-context-menu-props';
import EditLantanaDialog from '../dialogs/edit-lantana-dialog.vue';
import AddTagDialog from '../dialogs/add-tag-dialog.vue';
import AddTextDialog from '../dialogs/add-text-dialog.vue';
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';

const props = defineProps<LantanaContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
</script>