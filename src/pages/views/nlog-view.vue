<template>
    <NlogContextMenu />
</template>
<script setup lang="ts">
import type { NlogViewProps } from './nlog-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Nlog } from '@/classes/datas/nlog';
import { type Ref, ref } from 'vue';
import NlogContextMenu from './nlog-context-menu.vue';

const props = defineProps<NlogViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_nlog: Ref<Nlog> = ref(await props.nlog.clone())
</script>
