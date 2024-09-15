<template>
    <ShowPasswordResetLinkDialog />
</template>
<script setup lang="ts">
import type { ConfirmResetPasswordViewEmits } from './confirm-reset-password-view-emits';
import type { ConfirmResetPasswordViewProps } from './confirm-reset-password-view-props';
import ShowPasswordResetLinkDialog from '../dialogs/show-password-reset-link-dialog.vue';

const props = defineProps<ConfirmResetPasswordViewProps>();
const emits = defineEmits<ConfirmResetPasswordViewEmits>();
</script>
