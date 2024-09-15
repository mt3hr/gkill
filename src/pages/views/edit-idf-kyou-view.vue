<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditIDFKyouViewProps } from './edit-idf-kyou-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import KyouView from './kyou-view.vue';
import { IDFKyou } from '@/classes/datas/idf-kyou';
import { Kyou } from '@/classes/datas/kyou';

const props = defineProps<EditIDFKyouViewProps>();
const emits = defineEmits<KyouViewEmits>();

const cloned_idf_kyou: Ref<IDFKyou> = ref(await props.idf_kyou.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone())
</script>