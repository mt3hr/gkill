<template>
    <v-row>
        <v-col cols="auto" class="pb-0 mb-0">
            <SearchButton />
        </v-col>
        <v-spacer class="pa-0 ma-0"/>
        <v-col cols="auto" class="pb-0 mb-0">
            <ClearQueryButton />
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
