<template>
    <EditTimeIsDialog />
    <KyouHistoriesDialog />
    <ConfirmDeleteKyouDialog />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { AttachedTimeisPlaingContextMenuProps } from './attached-timeis-plaing-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { TimeIs } from '@/classes/datas/time-is';
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue';
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';

const props = defineProps<AttachedTimeisPlaingContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_timeis: Ref<TimeIs> = ref(await props.timeis.clone());
</script>
