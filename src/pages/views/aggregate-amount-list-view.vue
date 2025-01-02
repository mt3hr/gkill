<template>
    <v-card :ripple="false" :link="false">
        <v-virtual-scroll :items="aggregate_ammounts" :height="'50vh'" :width="200 + 8" ref="list_view">
            <template v-slot:default="{ item }">
                <AggregateAmountView :application_config="application_config" :gkill_api="gkill_api"
                    :aggregate_amount="item" :last_added_tag="last_added_tag" />
            </template>
        </v-virtual-scroll>
    </v-card>
</template>

<script lang="ts" setup>
import type { VVirtualScroll } from 'vuetify/components'
import type { AggregateAmountListViewProps } from './aggregate-amount-list-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import AggregateAmountView from './aggregate-amount-view.vue';

defineProps<AggregateAmountListViewProps>()
defineEmits<KyouViewEmits>()
</script>
<style lang="css"></style>