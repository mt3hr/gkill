<template>
    <KyouView />
    <tagHistoriesView />
</template>
<script setup lang="ts">
import type { TagHistoriesDialogProps } from './tag-histories-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import type { Kyou } from '@/classes/datas/kyou';
import type { TimeIs } from '@/classes/datas/time-is';
import { type Ref, ref } from 'vue';
import KyouView from '../views/kyou-view.vue';
import tagHistoriesView from '../views/tag-histories-view.vue';

const props = defineProps<TagHistoriesDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
const cloned_tag: Ref<TimeIs> = ref(await props.tag.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone())

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>