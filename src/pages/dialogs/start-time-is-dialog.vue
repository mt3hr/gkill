<template>
    <StartTimeIsView />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from '../views/kyou-view-emits';
import type { StartTimeIsDialogProps } from './start-time-is-dialog-props';
import StartTimeIsView from '../views/start-time-is-view.vue';

const props = defineProps<StartTimeIsDialogProps>();
const emits = defineEmits<KyouViewEmits>();
</script>
