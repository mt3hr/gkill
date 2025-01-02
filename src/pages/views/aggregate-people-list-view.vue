<template>
    <v-card :ripple="false" :link="false">
        <v-virtual-scroll :items="aggregate_peoples" :height="'50vh'" :width="200 + 8" ref="list_view">
            <template v-slot:default="{ item }">
                <AggregatePeopleView :application_config="application_config" :gkill_api="gkill_api"
                    :aggregate_people="item" :last_added_tag="last_added_tag" />
            </template>
        </v-virtual-scroll>
    </v-card>
</template>

<script lang="ts" setup>
import type { VVirtualScroll } from 'vuetify/components'
import type { KyouViewEmits } from './kyou-view-emits';
import type { AggregatePeopleListViewProps } from './aggregate-people-list-view-props';
import AggregatePeopleView from './aggregate-people-view.vue';

defineProps<AggregatePeopleListViewProps>()
defineEmits<KyouViewEmits>()
</script>
<style lang="css"></style>