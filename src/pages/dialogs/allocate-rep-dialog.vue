<template>
    <AllocateRepView />
</template>
<script lang="ts" setup>
import type { AllocateRepDialogEmits } from './allocate-rep-dialog-emits';
import type { AllocateRepDialogProps } from './allocate-rep-dialog-props';
import AllocateRepView from '../views/allocate-rep-view.vue';

const props = defineProps<AllocateRepDialogProps>();
const emits = defineEmits<AllocateRepDialogEmits>();
</script>
