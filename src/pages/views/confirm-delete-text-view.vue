<template>
    <KyouView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ConfirmDeleteTextViewProps } from './confirm-delete-text-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { Text } from '@/classes/datas/text';
import { Kyou } from '@/classes/datas/kyou';
import KyouView from './kyou-view.vue';

const props = defineProps<ConfirmDeleteTextViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_text: Ref<Text> = ref(await props.text.clone());
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone());
</script>
