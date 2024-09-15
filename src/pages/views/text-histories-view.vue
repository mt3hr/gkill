<template>
    <TextView />
</template>
<script lang="ts" setup>
import { type Ref, computed, ref } from 'vue';
import type { KyouViewEmits } from './kyou-view-emits';
import type { TextHistoriesViewProps } from './text-histories-view-props';
import TextView from './text-view.vue';
import { Text } from '@/classes/datas/text';

const props = defineProps<TextHistoriesViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_text: Ref<Text> = ref(await props.text.clone());
const cloned_text_histories = computed(async () => {
    const errors = await cloned_text.value.load_attached_histories()
    if (errors && errors.length !== 0) {
        emits('received_errors', errors)
        return
    }
    return cloned_text.value.attached_histories
});
</script>
