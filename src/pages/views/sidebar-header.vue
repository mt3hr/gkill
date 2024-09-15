<template>
    <ClearQueryButton />
    <SearchButton />
</template>
<script setup lang="ts">
import ClearQueryButton from './clear-query-button.vue';
import SearchButton from './search-button.vue';
import type { SidebarHeaderEmits } from './sidebar-header-emits';
import type { SidebarHeaderProps } from './sidebar-header-props';

const props = defineProps<SidebarHeaderProps>();
const emits = defineEmits<SidebarHeaderEmits>();
</script>
