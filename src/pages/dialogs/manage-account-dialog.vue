<template>
    <ManageAccountView />
</template>
<script lang="ts" setup>
import type { ManageAccountDialogEmits } from './manage-account-dialog-emits';
import type { ManageAccountDialogProps } from './manage-account-dialog-props';
import ManageAccountView from '../views/manage-account-view.vue';

const props = defineProps<ManageAccountDialogProps>();
const emits = defineEmits<ManageAccountDialogEmits>();
</script>