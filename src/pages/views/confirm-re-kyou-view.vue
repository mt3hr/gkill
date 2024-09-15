<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ConfirmReKyouViewProps } from './confirm-re-kyou-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { Kyou } from '@/classes/datas/kyou';
import KyouView from './kyou-view.vue';

const props = defineProps<ConfirmReKyouViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>
