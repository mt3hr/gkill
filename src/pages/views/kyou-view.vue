<template>
    <AttachedTag v-for="attached_tag, index in cloned_kyou.attached_tags" />
    <AttachedText v-for="attached_text, index in cloned_kyou.attached_texts" />
    <AttachedTimeIsPlaing v-for="attached_timeis_plaing, index in cloned_kyou.attached_timeis" />
    <kyouDialog />
    <EditKmemoDialog />
    <EditMiDialog />
    <EditNlogDialog />
    <EditLantanaDialog />
    <EditTimeIsDialog />
    <EditURLogDialog />
    <EditIDFKyouDialog />
    <ConfirmDeleteKyouDialog />
    <KyouHistoriesDialog />
    <AddTagDialog />
    <EditTagDialog />
    <ConfirmDeleteTagDialog />
    <TagHistoriesDialog />
    <AddTextDialog />
    <EditTextDialog />
    <ConfirmDeleteTextDialog />
    <TextHistoriesDialog />
    <KmemoView />
    <MiView />
    <NlogView />
    <LantanaView />
    <TimeIsView />
    <URLogView />
    <IDFKyouView />
    <ReKyouView />
    <GitCommitLogView />
</template>
<script setup lang="ts">
import type { Kyou } from '@/classes/datas/kyou';
import { type Ref, ref } from 'vue';

import AddTagDialog from '../dialogs/add-tag-dialog.vue';
import AddTextDialog from '../dialogs/add-text-dialog.vue';
import AttachedTag from './attached-tag.vue';
import AttachedText from './attached-text.vue';
import AttachedTimeIsPlaing from './attached-time-is-plaing.vue';
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue';
import ConfirmDeleteTagDialog from '../dialogs/confirm-delete-tag-dialog.vue';
import ConfirmDeleteTextDialog from '../dialogs/confirm-delete-text-dialog.vue';
import EditIDFKyouDialog from '../dialogs/edit-idf-kyou-dialog.vue';
import EditKmemoDialog from '../dialogs/edit-kmemo-dialog.vue';
import EditLantanaDialog from '../dialogs/edit-lantana-dialog.vue';
import EditMiDialog from '../dialogs/edit-mi-dialog.vue';
import EditNlogDialog from '../dialogs/edit-nlog-dialog.vue';
import EditTagDialog from '../dialogs/edit-tag-dialog.vue';
import EditTextDialog from '../dialogs/edit-text-dialog.vue';
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue';
import EditURLogDialog from '../dialogs/edit-ur-log-dialog.vue';
import GitCommitLogView from './git-commit-log-view.vue';
import IDFKyouView from './idf-kyou-view.vue';
import KmemoView from './kmemo-view.vue';
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue';
import LantanaView from './lantana-view.vue';
import MiView from './mi-view.vue';
import NlogView from './nlog-view.vue';
import ReKyouView from './re-kyou-view.vue';
import TagHistoriesDialog from '../dialogs/tag-histories-dialog.vue';
import TextHistoriesDialog from '../dialogs/text-histories-dialog.vue';
import TimeIsView from './time-is-view.vue';
import URLogView from './ur-log-view.vue';
import kyouDialog from '../dialogs/kyou-dialog.vue';

import type { KyouViewEmits } from './kyou-view-emits';
import type { KyouViewProps } from './kyou-view-props';

const props = defineProps<KyouViewProps>();
const emits = defineEmits<KyouViewEmits>();

const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());

async function load_attached_infos(): Promise<void> {
    throw new Error('Not implemented');
}
async function clear_attached_infos(): Promise<void> {
    throw new Error('Not implemented');
}
async function update_check(is_checked: boolean): Promise<void> {
    throw new Error('Not implemented');
}
</script>
<style scoped></style>
