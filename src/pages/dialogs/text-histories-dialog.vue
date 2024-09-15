<template>
    <KyouView />
    <TextHistoriesView />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import type { TextHistoriesDialogProps } from './text-histories-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import type { Kyou } from '@/classes/datas/kyou';
import { Text } from '@/classes/datas/text';
import KyouView from '../views/kyou-view.vue';
import TextHistoriesView from '../views/text-histories-view.vue';

const props = defineProps<TextHistoriesDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
const cloned_text: Ref<Text> = ref(await props.text.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone())
async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>