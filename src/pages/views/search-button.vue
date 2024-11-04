<template>
    <v-btn @click="emits('request_search')" hide-details class="pa-0 ma-0">検索</v-btn>
</template>
<script lang="ts" setup>
import type { SearchButtonEmits } from './search-button-emits'

const emits = defineEmits<SearchButtonEmits>()
</script>
