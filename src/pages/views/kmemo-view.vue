<template>
    <KmemoContextMenu />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import KmemoContextMenu from './kmemo-context-menu.vue';
import type { Kmemo } from '@/classes/datas/kmemo';
import type { KmemoViewProps } from './kmemo-view-props.ts';
import type { KyouViewEmits } from './kyou-view-emits';

const props = defineProps<KmemoViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_kmemo: Ref<Kmemo> = ref(await props.kmemo.clone());
</script>
