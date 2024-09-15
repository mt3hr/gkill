<template>
    <EditIdfKyouDialog />
    <AddTagDialog />
    <AddTextDialog />
    <ConfirmDeleteKyouDialog />
    <ConfirmReKyouDialog />
</template>
<script lang="ts" setup>
import type { IDFKyouContextMenuProps } from './idf-kyou-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import EditIdfKyouDialog from '../dialogs/edit-idf-kyou-dialog.vue';
import AddTagDialog from '../dialogs/add-tag-dialog.vue';
import AddTextDialog from '../dialogs/add-text-dialog.vue';
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';

const props = defineProps<IDFKyouContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
</script>
