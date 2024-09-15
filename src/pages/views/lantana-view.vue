<template>
    <LantanaContextMenu />
</template>
<script setup lang="ts">
import type { LantanaViewProps } from './lantana-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Lantana } from '@/classes/datas/lantana';
import { type Ref, ref } from 'vue';
import LantanaContextMenu from './lantana-context-menu.vue';

const props = defineProps<LantanaViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_lantana: Ref<Lantana> = ref(await props.lantana.clone());
</script>
