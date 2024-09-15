<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditReKyouViewProps } from './edit-re-kyou-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { ReKyou } from '@/classes/datas/re-kyou';
import KyouView from './kyou-view.vue';

const props = defineProps<EditReKyouViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_rekyou: Ref<ReKyou> = ref(await props.rekyou.clone());
</script>