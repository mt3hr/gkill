<template>
    <AddNewFoloderView />
</template>
<script lang="ts" setup>
import type { AddNewFoloderDialogEmits } from './add-new-foloder-dialog-emits';
import type { AddNewFoloderDialogProps } from './add-new-foloder-dialog-props';
import AddNewFoloderView from '../views/add-new-foloder-view.vue';

const props = defineProps<AddNewFoloderDialogProps>();
const emits = defineEmits<AddNewFoloderDialogEmits>();
</script>