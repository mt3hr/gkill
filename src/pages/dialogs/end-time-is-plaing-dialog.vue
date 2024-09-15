<template>
    <EndTimeIsPlaingView />
</template>
<script lang="ts" setup>
import EndTimeIsPlaingView from '../views/end-time-is-plaing-view.vue';
import type { EndTimeIsPlaingDialogProps } from './end-time-is-plaing-dialog-props';
import type { KyouViewEmits } from '../views/kyou-view-emits';

const props = defineProps<EndTimeIsPlaingDialogProps>();
const emits = defineEmits<KyouViewEmits>();
</script>