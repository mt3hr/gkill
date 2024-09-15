<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { KyouDialogProps } from './kyou-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import type { Kyou } from '@/classes/datas/kyou';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<KyouDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
