<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditNlogViewProps } from './edit-nlog-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import KyouView from './kyou-view.vue';
import { Kyou } from '@/classes/datas/kyou';
import { Nlog } from '@/classes/datas/nlog';

const props = defineProps<EditNlogViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_nlog: Ref<Nlog> = ref(await props.nlog.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
kyou_view: KyouView;
</script>
