<template>
    <LantanaFlower />
    <LantanaTextView />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import type { LantanaFlowersViewEmits } from './lantana-flowers-view-emits';
import type { LantanaFlowersViewProps } from './lantana-flowers-view-props';
import LantanaFlower from './lantana-flower.vue';
import LantanaTextView from './lantana-text-view.vue';
import { LantanaFlowerState } from '@/classes/lantana/lantana-flower-state';

const props = defineProps<LantanaFlowersViewProps>();
const emits = defineEmits<LantanaFlowersViewEmits>();

const flower_state_1: Ref<LantanaFlowerState> = ref(LantanaFlowerState.none);
const flower_state_2: Ref<LantanaFlowerState> = ref(LantanaFlowerState.none);
const flower_state_3: Ref<LantanaFlowerState> = ref(LantanaFlowerState.none);
const flower_state_4: Ref<LantanaFlowerState> = ref(LantanaFlowerState.none);
const flower_state_5: Ref<LantanaFlowerState> = ref(LantanaFlowerState.none);

async function get_mood(): Promise<Number> {
    throw new Error('Not implemented');
}
async function set_mood(mood: Number): Promise<void> {
    throw new Error('Not implemented');
}
async function emit_updated_mood(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
