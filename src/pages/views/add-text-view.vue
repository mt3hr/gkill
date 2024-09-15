<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { AddTextViewProps } from './add-text-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Kyou } from '@/classes/datas/kyou';
import { Text } from '@/classes/datas/text';
import KyouView from './kyou-view.vue';

const props = defineProps<AddTextViewProps>();
const emits = defineEmits<KyouViewEmits>();
const text: Ref<Text> = ref(await props.text.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>