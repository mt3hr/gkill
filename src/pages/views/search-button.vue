<template>
    <v-btn @click="emits('requested_search')" @dblclick="emits('requested_search_with_update_cache')" hide-details class="pa-0 ma-0" color="primary">検索</v-btn>
</template>
<script lang="ts" setup>
import type { SearchButtonEmits } from './search-button-emits'

const emits = defineEmits<SearchButtonEmits>()
</script>
