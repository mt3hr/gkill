<template>
    <v-btn @click="emits('request_clear_find_query')" hide-details class="pb-0 mb-0">クリア</v-btn>
</template>
<script lang="ts" setup>
import type { ClearButtonEmits } from './clear-button-emits'
const emits = defineEmits<ClearButtonEmits>()
</script>
<style lang="css">
</style>
