<template>
    <div class="line_label" v-for="row, index in line_label_data.lines" :key="index" :style="style">
        <p v-if="index === 0">
            {{ line_label_data.label }}
        </p>
    </div>
</template>
<script setup lang="ts">
import type { KFTLLineLabelProps } from './kftl-line-label-props'

defineProps<KFTLLineLabelProps>()
</script>
<style scoped>
.line_label {
    height: 24px;
    font-size: 1em;
}
</style>