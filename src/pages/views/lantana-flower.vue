<template>
    <img class="lantana_icon_harf_right" />
    <img class="lantana_icon_harf_left" />
    <img class="lantana_icon_full_left" />
    <img class="lantana_icon_harf_left_gray" />
    <img class="lantana_icon_harf_right_gray" />
    <img class="lantana_icon_full_left_gray" />
</template>
<script lang="ts" setup>
import type { LantanaFlowerEmits } from './lantana-flower-emits';
import type { LantanaFlowerProps } from './lantana-flower-props';

const props = defineProps<LantanaFlowerProps>();
const emits = defineEmits<LantanaFlowerEmits>();
</script>
