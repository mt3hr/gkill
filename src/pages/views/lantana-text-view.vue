<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { LantanaTextTypeSelectBox } from './lantana-text-type-select-box.vue';
import { LantanaTextViewEmits } from './lantana-text-view-emits';

// ˄

export class LantanaTextView {
    // ˅
    
    // ˄

    private lantana_text: Ref<LantanaTextData>;

    private text_data: Ref<string>;

    private emits: LantanaTextViewEmits;

    private lantanaTextTypeSelectBox: LantanaTextTypeSelectBox;

    private async emit_request_delete_lantana_text(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async emit_update_lantana_text(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async update_text_content(text_content: string): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async update_text_type(text_type: LantanaTextType): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async get_text_data(): Promise<LantanaTextData> {
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