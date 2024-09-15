<template>
    <EditRepTypeView />
</template>
<script lang="ts" setup>
import type { EditRepTypeDialogEmits } from './edit-rep-type-dialog-emits';
import type { EditRepTypeDialogProps } from './edit-rep-type-dialog-props';
import EditRepTypeView from '../views/edit-rep-type-view.vue';

const props = defineProps<EditRepTypeDialogProps>();
const emits = defineEmits<EditRepTypeDialogEmits>();
</script>
