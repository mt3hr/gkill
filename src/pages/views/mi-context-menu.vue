<template>
    <EditMiDialog />
    <AddTagDialog />
    <AddTextDialog />
    <ConfirmReKyouDialog />
    <ConfirmDeleteKyouDialog />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits';
import type { MiContextMenuProps } from './mi-context-menu-props';
import EditMiDialog from '../dialogs/edit-mi-dialog.vue';
import AddTagDialog from '../dialogs/add-tag-dialog.vue';
import AddTextDialog from '../dialogs/add-text-dialog.vue';
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';

const props = defineProps<MiContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
</script>
