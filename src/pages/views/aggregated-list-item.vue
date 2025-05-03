<template>
    <v-card class="pa-0 ma-0 aggregated_list_item">
        <div>
            {{ aggregated_item.title }}
        </div>
        <v-row class="pa-0 ma-0">
            <v-col class="pa-0 ma-0" cols="auto">
                {{ aggregated_item.value }}
            </v-col>
        </v-row>
        <!-- //TODO matchKyousのダイアログ -->
    </v-card>
</template>
<script lang="ts" setup>
import type AggregatedListItemProps from './aggregated-list-item-props';

defineProps<AggregatedListItemProps>()
</script>
<style lang="css" scoped>
.aggregated_list_item {
    border-top: 1px solid silver;
}
</style>