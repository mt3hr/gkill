<template>
    <AddRepView />
</template>
<script lang="ts" setup>
import type { AddRepDialogEmits } from './add-rep-dialog-emits';
import type { AddRepDialogProps } from './add-rep-dialog-props';
import AddRepView from '../views/add-rep-view.vue';

const props = defineProps<AddRepDialogProps>();
const emits = defineEmits<AddRepDialogEmits>();
</script>
