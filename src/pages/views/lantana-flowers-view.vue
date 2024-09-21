<template>
    <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
        :state="flower_state_1" @clicked_left="set_mood(1)" @clicked_right="set_mood(2)" />
    <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
        :state="flower_state_2" @clicked_left="set_mood(3)" @clicked_right="set_mood(4)" />
    <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
        :state="flower_state_3" @clicked_left="set_mood(5)" @clicked_right="set_mood(6)" />
    <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
        :state="flower_state_4" @clicked_left="set_mood(7)" @clicked_right="set_mood(8)" />
    <LantanaFlower :application_config="application_config" :editable="editable" :gkill_api="gkill_api"
        :state="flower_state_5" @clicked_left="set_mood(9)" @clicked_right="set_mood(10)" />
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
defineExpose({ set_mood, get_mood })

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
