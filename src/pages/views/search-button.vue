<template>
    <v-btn dark @click="emits('requested_search')" @dblclick="emits('requested_search_with_update_cache')" hide-details
        class="pa-0 ma-0" color="primary">{{ $t("SEARCH_TITLE") }}</v-btn>
</template>
<script lang="ts" setup>
import type { SearchButtonEmits } from './search-button-emits'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const emits = defineEmits<SearchButtonEmits>()
</script>
