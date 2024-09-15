<template>
    <DnotePeopleView />
</template>
<script setup lang="ts">
import type { DnotePeoplesListViewEmits } from './dnote-peoples-list-view-emits';
import type { DnotePeoplesListViewProps } from './dnote-peoples-list-view-props';
import type { Kyou } from '@/classes/datas/kyou';
import { type Ref, ref } from 'vue';
import DnotePeopleView from './dnote-people-view.vue';

const props = defineProps<DnotePeoplesListViewProps>();
const emits = defineEmits<DnotePeoplesListViewEmits>();
const etxructed_sorted_people_kyous: Ref<Array<Kyou>> = ref(props.timeis_kyous.concat());
</script>
