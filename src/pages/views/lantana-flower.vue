<template>
    <div class="lantana_icon">
        <img v-if="state == LantanaFlowerState.full" class="lantana_icon_harf_left" :src="lantana_icon"
            @click="emit_clicked_left" />
        <img v-if="state == LantanaFlowerState.full" class="lantana_icon_harf_right" :src="lantana_icon"
            @click="emit_clicked_right" />
        <img v-if="state == LantanaFlowerState.half" class="lantana_icon_harf_left" :src="lantana_icon"
            @click="emit_clicked_left" />
        <img v-if="state == LantanaFlowerState.half" class="lantana_icon_harf_right gray" :src="lantana_icon"
            @click="emit_clicked_right" />
        <img v-if="state == LantanaFlowerState.none" class="lantana_icon_harf_left gray" :src="lantana_icon"
            @click="emit_clicked_left" />
        <img v-if="state == LantanaFlowerState.none" class="lantana_icon_harf_right gray" :src="lantana_icon"
            @click="emit_clicked_right" />
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { LantanaFlowerEmits } from './lantana-flower-emits'
import type { LantanaFlowerProps } from './lantana-flower-props'
import { LantanaFlowerState } from '@/classes/lantana/lantana-flower-state';
import lantana_icon from '/public/lantana_icon.png'

const props = defineProps<LantanaFlowerProps>()
const emits = defineEmits<LantanaFlowerEmits>()

function emit_clicked_left() {
    if (props.editable) {
        emits('clicked_left')
    }
}
function emit_clicked_right() {
    if (props.editable) {
        emits('clicked_right')
    }
}
</script>

<style scoped>
.lantana_icon {
    position: relative;
    width: 50px;
    height: 50px;
    max-width: 50px;
    min-width: 50px;
    max-height: 50px;
    min-height: 50px;
    display: inline-block;
}

.lantana_icon_fill {
    width: 50px;
    height: 50px;
    max-width: 50px;
    min-width: 50px;
    max-height: 50px;
    min-height: 50px;
    display: inline-block;
    z-index: 10;
}

.lantana_icon_harf_left {
    position: absolute;
    left: 0px;
    width: 25px;
    height: 50px;
    max-width: 25px;
    min-width: 25px;
    max-height: 50px;
    min-height: 50px;
    object-fit: cover;
    object-position: 0 0;
    display: inline-block;
    z-index: 10;
}

.lantana_icon_harf_right {
    position: absolute;
    left: 0px;
    width: 50px;
    height: 50px;
    max-width: 50px;
    min-width: 50px;
    max-height: 50px;
    min-height: 50px;
    display: inline-block;
    z-index: 9;
}

.lantana_icon_none {
    width: 50px;
    height: 50px;
    max-width: 50px;
    min-width: 50px;
    max-height: 50px;
    min-height: 50px;
    display: inline-block;
    z-index: 10;
}

.gray {
    filter: grayscale(100%);
}
</style>