<template>
    <URLogContextMenu />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import type { URLogViewProps } from './ur-log-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import URLogContextMenu from './ur-log-context-menu.vue';
import type { URLog } from '@/classes/datas/ur-log';

const props = defineProps<URLogViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_urlog: Ref<URLog> = ref(await props.urlog.clone());
</script>
