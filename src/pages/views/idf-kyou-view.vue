<template>
    <IDFKyouContextMenu />
</template>
<script setup lang="ts">
import IDFKyouContextMenu from './idf-kyou-context-menu.vue';
import type { IDFKyou } from '@/classes/datas/idf-kyou';
import { type Ref, ref } from 'vue';
import type { IDFKyouProps } from './idf-kyou-props';
import type { KyouViewEmits } from './kyou-view-emits';

const props = defineProps<IDFKyouProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_idf_kyou: Ref<IDFKyou> = ref(await props.idf_kyou.clone());

</script>
