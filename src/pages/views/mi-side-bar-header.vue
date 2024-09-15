<template>
    <ClearQueryButton />
    <SearchButton />
</template>
<script lang="ts" setup>
import ClearQueryButton from './clear-query-button.vue';
import SearchButton from './search-button.vue';
import type { miSidebarHeaderEmits } from './mi-sidebar-header-emits';
import type { miSidebarHeaderProps } from './mi-sidebar-header-props';
const emits = defineEmits<miSidebarHeaderEmits>();
const props = defineProps<miSidebarHeaderProps>();
</script>