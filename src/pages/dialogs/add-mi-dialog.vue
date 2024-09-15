<template>
    <AddMiView />
</template>
<script setup lang="ts">
import AddMiView from '../views/add-mi-view.vue';
import type { AddMiDialogProps } from './add-mi-dialog-props';
import type { KyouViewEmits } from '../views/kyou-view-emits';

const props = defineProps<AddMiDialogProps>();
const emits = defineEmits<KyouViewEmits>();
</script>
<style scoped></style>
