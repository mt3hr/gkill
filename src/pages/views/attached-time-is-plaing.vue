<template>
    <EditTimeIsDialog />
    <AttachedTimeisPlaingContextMenu />
    <ConfirmDeleteKyouDialog />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import type { TimeIs } from '@/classes/datas/time-is';
import type { AttachedTimeIsPlaingProps } from './attached-time-is-plaing-props';
import type { KyouViewEmits } from './kyou-view-emits';
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue';
import AttachedTimeisPlaingContextMenu from './attached-timeis-plaing-context-menu.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';

const props = defineProps<AttachedTimeIsPlaingProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_timeis: Ref<TimeIs> = ref(await props.timeis.clone());
</script>