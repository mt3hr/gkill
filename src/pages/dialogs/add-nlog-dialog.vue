<template>
    <AddNlogView />
</template>
<script lang="ts" setup>
import AddNlogView from '../views/add-nlog-view.vue';
import type { AddNlogDialogProps } from './add-nlog-dialog-props';
import type { KyouViewEmits } from '../views/kyou-view-emits';

const props = defineProps<AddNlogDialogProps>();
const emits = defineEmits<KyouViewEmits>();
</script>
