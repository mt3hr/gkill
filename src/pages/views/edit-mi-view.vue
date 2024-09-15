<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditMiViewProps } from './edit-mi-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Kyou } from '@/classes/datas/kyou';
import type { Mi } from '@/classes/datas/mi';
import KyouView from './kyou-view.vue';

const props = defineProps<EditMiViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_mi: Ref<Mi> = ref(await props.mi.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>