<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditKmemoViewProps } from './edit-kmemo-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import KyouView from './kyou-view.vue';
import { Kmemo } from '@/classes/datas/kmemo';
import { Kyou } from '@/classes/datas/kyou';

const props = defineProps<EditKmemoViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_kmemo: Ref<Kmemo> = ref(await props.kmemo.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>