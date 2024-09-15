<template>
    <ConfirmResetPasswordView />
</template>
<script lang="ts" setup>
import type { ConfirmResetPasswordDialogEmits } from './confirm-reset-password-dialog-emits';
import type { ConfirmResetPasswordDialogProps } from './confirm-reset-password-dialog-props';
import ConfirmResetPasswordView from '../views/confirm-reset-password-view.vue';

const props = defineProps<ConfirmResetPasswordDialogProps>();
const emits = defineEmits<ConfirmResetPasswordDialogEmits>();

</script>