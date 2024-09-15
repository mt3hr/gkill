<template>
    <ConfirmDeleteRepView />
</template>
<script lang="ts" setup>
import type { ConfirmDeleteRepDialogEmits } from './confirm-delete-rep-dialog-emits';
import type { ConfirmDeleteRepDialogProps } from './confirm-delete-rep-dialog-props';
import ConfirmDeleteRepView from '../views/confirm-delete-rep-view.vue';

const props = defineProps<ConfirmDeleteRepDialogProps>();
const emits = defineEmits<ConfirmDeleteRepDialogEmits>();
</script>