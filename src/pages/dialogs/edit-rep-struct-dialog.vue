<template>
    <EditRepStructView />
</template>
<script lang="ts" setup>
import type { EditRepStructDialogEmits } from './edit-rep-struct-dialog-emits';
import type { EditRepStructDialogProps } from './edit-rep-struct-dialog-props';
import EditRepStructView from '../views/edit-rep-struct-view.vue';

const props = defineProps<EditRepStructDialogProps>();
const emits = defineEmits<EditRepStructDialogEmits>();
</script>