<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { LantanaFlower } from './lantana-flower.vue';
import { LantanaFlowersViewEmits } from './lantana-flowers-view-emits';
import { LantanaFlowersViewProps } from './lantana-flowers-view-props';
import { LantanaTextView } from './lantana-text-view';

// ˄

export class LantanaFlowersView {
    // ˅
    
    // ˄

    private flower_state_1: Ref<LantanaFlowerState>;

    private flower_state_2: Ref<LantanaFlowerState>;

    private flower_state_3: Ref<LantanaFlowerState>;

    private flower_state_4: Ref<LantanaFlowerState>;

    private flower_state_5: Ref<LantanaFlowerState>;

    private lantanaFlower: Array<LantanaFlower>;

    private props: LantanaFlowersViewProps;

    private emits: LantanaFlowersViewEmits;

    private lantanaTextView: LantanaTextView;

    async get_mood(): Promise<Number> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_mood(mood: Number): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async emit_updated_mood(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>