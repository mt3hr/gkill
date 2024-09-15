<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { KyouHistoriesViewProps } from './kyou-histories-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import KyouView from './kyou-view.vue';
import { Kyou } from '@/classes/datas/kyou';

const props = defineProps<KyouHistoriesViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>