<template>
    <v-row class="py-0 my-0">
        <v-col cols="auto" class="py-0 my-0 background-white">
            <SearchButton @requested_search="emits('requested_search')" @requested_search_with_update_cache="emits('requested_search_with_update_cache')"/>
        </v-col>
        <v-spacer class="pa-0 ma-0 background-white" />
        <v-col cols="auto" class="py-0 my-0 background-white">
            <ClearQueryButton @requested_clear_find_query="emits('requested_clear_find_query')" />
        </v-col>
    </v-row>
</template>
<script setup lang="ts">
import ClearQueryButton from './clear-query-button.vue'
import SearchButton from './search-button.vue'
import type { SidebarHeaderEmits } from './sidebar-header-emits'
import type { SidebarHeaderProps } from './sidebar-header-props'

const props = defineProps<SidebarHeaderProps>()
const emits = defineEmits<SidebarHeaderEmits>()
</script>
<style lang="css" scoped>
.background-white {
    background-color: white;
    z-index: 10000;
}
</style>