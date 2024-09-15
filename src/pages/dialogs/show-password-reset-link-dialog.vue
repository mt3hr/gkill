<template>
    <ShowPasswordResetLinkView />
</template>
<script lang="ts" setup>
import type { ShowPasswordResetLinkDialogEmits } from './show-password-reset-link-dialog-emits';
import type { ShowPasswordResetLinkDialogProps } from './show-password-reset-link-dialog-props';
import ShowPasswordResetLinkView from '../views/show-password-reset-link-view.vue';

const props = defineProps<ShowPasswordResetLinkDialogProps>();
const emits = defineEmits<ShowPasswordResetLinkDialogEmits>();
</script>
