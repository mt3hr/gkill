<template>
    <KyouDialog />
</template>
<script setup lang="ts">
import type { DnotePeopleViewEmits } from './dnote-people-view-emits';
import type { DnotePeopleViewProps } from './dnote-people-view-props';
import KyouDialog from '../dialogs/kyou-dialog.vue';

const props = defineProps<DnotePeopleViewProps>();
const emits = defineEmits<DnotePeopleViewEmits>();
</script>
