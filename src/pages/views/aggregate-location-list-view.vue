<template>
    <v-card :ripple="false" :link="false">
        <v-virtual-scroll class="aggregate_location_list" :items="aggregate_locations" :height="'50vh'" :width="200 + 8"
            ref="list_view">
            <template v-slot:default="{ item }">
                <AggregateLocationView :application_config="application_config" :gkill_api="gkill_api"
                    :aggregate_location="item" :last_added_tag="last_added_tag" />
            </template>
        </v-virtual-scroll>
    </v-card>
</template>

<script lang="ts" setup>
import type { VVirtualScroll } from 'vuetify/components'
import type { KyouViewEmits } from './kyou-view-emits';
import AggregateLocationView from './aggregate-location-view.vue';
import type { AggregateLocationListViewProps } from './aggregate-location-list-view-props';

defineProps<AggregateLocationListViewProps>()
defineEmits<KyouViewEmits>()
</script>
<style lang="css"></style>