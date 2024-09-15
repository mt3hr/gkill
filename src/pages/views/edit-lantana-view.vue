<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditLantanaViewProps } from './edit-lantana-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { Lantana } from '@/classes/datas/lantana';
import { Kyou } from '@/classes/datas/kyou';
import KyouView from './kyou-view.vue';

const props = defineProps<EditLantanaViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_lantana: Ref<Lantana> = ref(await props.lantana.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>
