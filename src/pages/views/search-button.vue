<template>
    <v-btn dark @click="emits('requested_search')" v-long-press="() => emits('requested_search_with_update_cache')"
        hide-details class="pa-0 ma-0" color="primary">{{ i18n.global.t("SEARCH_TITLE") }}</v-btn>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { SearchButtonEmits } from './search-button-emits'

const emits = defineEmits<SearchButtonEmits>()
</script>
