<template>
    <CreateAccountView />
</template>
<script lang="ts" setup>
import type { CreateAccountDialogEmits } from './create-account-dialog-emits';
import type { CreateAccountDialogProps } from './create-account-dialog-props';
import CreateAccountView from '../views/create-account-view.vue';

const props = defineProps<CreateAccountDialogProps>();
const emits = defineEmits<CreateAccountDialogEmits>();
</script>
