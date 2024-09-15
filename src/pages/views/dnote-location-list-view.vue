<template>
    <DnoteLocationView />
</template>
<script lang="ts" setup>
import type { DnoteLocationListViewEmits } from './dnote-location-list-view-emits';
import type { DnoteLocationListViewProps } from './dnote-location-list-view-props';
import type { Kyou } from '@/classes/datas/kyou';
import { type Ref, ref } from 'vue';
import DnoteLocationView from './dnote-location-view.vue';

const props = defineProps<DnoteLocationListViewProps>();
const emits = defineEmits<DnoteLocationListViewEmits>();
const etxructed_sorted_location_kyous: Ref<Array<Kyou>> = ref(props.timeis_kyous);
</script>