<template>
    <KyouView />
</template>
<script setup lang="ts">
import type { KyouListViewEmits } from './kyou-list-view-emits';
import type { KyouListViewProps } from './kyou-list-view-props';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import { GkillError } from '@/classes/api/gkill-error';
import { Kyou } from '@/classes/datas/kyou';
import { type Ref, ref } from 'vue';
import KyouView from './kyou-view.vue';

const props = defineProps<KyouListViewProps>();
const emits = defineEmits<KyouListViewEmits>();
const cloned_find_query: Ref<FindKyouQuery> = ref(await props.query.clone());
const match_kyous: Array<Kyou> = ref(props.matched_kyous.concat());
const is_loading: Ref<boolean> = ref(false);
const scroll_distance_from_top_px: Ref<Number> = ref(0);
const checked_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>());

async function reload(): Promise<Array<GkillError>> {
    throw new Error('Not implemented');
}
async function scroll_to_kyou(kyou: Kyou): Promise<boolean> {
    throw new Error('Not implemented');
}
async function scroll_to_time(time: Date): Promise<boolean> {
    throw new Error('Not implemented');
}
</script>
