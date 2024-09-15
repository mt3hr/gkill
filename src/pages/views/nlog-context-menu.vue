<template>
    <EditNlogDialog />
    <AddTagDialog />
    <AddTextDialog />
    <ConfirmDeleteKyouDialog />
    <ConfirmReKyouDialog />
</template>
<script lang="ts" setup>

import type { KyouViewEmits } from './kyou-view-emits';
import type { NlogContextMenuProps } from './nlog-context-menu-props';
import EditNlogDialog from '../dialogs/edit-nlog-dialog.vue';
import AddTagDialog from '../dialogs/add-tag-dialog.vue';
import AddTextDialog from '../dialogs/add-text-dialog.vue';
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';

const props = defineProps<NlogContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
</script>
