<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditTimeIsViewProps } from './edit-time-is-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { Kyou } from '@/classes/datas/kyou';
import { TimeIs } from '@/classes/datas/time-is';
import KyouView from './kyou-view.vue';

const props = defineProps<EditTimeIsViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_timeis: Ref<TimeIs> = ref(await props.timeis.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>