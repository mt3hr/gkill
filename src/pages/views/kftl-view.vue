<template>
    <KyouListView />
    <KFTLLineLabel />
    <KFTLTemplateDialog />
</template>

<script setup lang="ts">
import { computed, ref, type Ref } from 'vue';
import type { GkillError } from '@/classes/api/gkill-error';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import { FindTimeIsQuery } from '@/classes/api/find_query/find-time-is-query';
import { LineLabelData } from '@/classes/kftl/line-label-data';

import type { KFTLProps } from './kftl-props';
import type { KFTLViewEmits } from './kftl-view-emits';

import KyouListView from './kyou-list-view.vue';
import KFTLLineLabel from './kftl-line-label.vue';
import KFTLTemplateDialog from '../dialogs/kftl-template-dialog.vue';

const text_area_content: Ref<string> = ref("");
const text_area_width: Ref<Number> = ref(0);
const text_area_width_px = computed(() => text_area_width.value.toString().concat("px"))
const text_area_height: Ref<Number> = ref(0);
const text_area_height_px = computed(() => text_area_height.value.toString().concat("px"))
const line_label_width: Ref<Number> = ref(0);
const line_label_width_px = computed(() => line_label_width.value.toString().concat("px"))
const line_label_height: Ref<Number> = ref(0);
const line_label_height_px = computed(() => line_label_height.value.toString().concat("px"))

const line_label_datas: Ref<Array<LineLabelData>> = ref(new Array<LineLabelData>());
const line_label_styles: Ref<Array<string>> = ref(new Array<string>());
const invalid_line_numbers: Ref<Array<Number>> = ref(new Array<Number>());
const is_requested_submit: Ref<boolean> = ref(false);
const find_kyou_query_plaing_timeis: Ref<FindKyouQuery> = ref(new FindKyouQuery());

const props = defineProps<KFTLProps>();
const emits = defineEmits<KFTLViewEmits>();

async function restore_content_from_localstorage(): Promise<void> {
    //TODO
    throw new Error('Not implemented');
}

async function save_content_to_localstorage(): Promise<void> {
    //TODO
    throw new Error('Not implemented');
}

async function update_line_labels(): Promise<void> {
    //TODO
    throw new Error('Not implemented');
}

async function is_invalid_line(line_index: Number): Promise<boolean> {
    //TODO
    throw new Error('Not implemented');
}

async function submit(): Promise<Array<GkillError>> {
    //TODO
    throw new Error('Not implemented');
}

async function clear(): Promise<void> {
    //TODO
    throw new Error('Not implemented');
}

async function show_kftl_template_dialog(): Promise<void> {
    //TODO
    throw new Error('Not implemented');
}

async function resize(): Promise<void> {
    //TODO
    throw new Error('Not implemented');
}

async function apply_application_config(): Promise<Array<GkillError>> {
    //TODO
    throw new Error('Not implemented');
}

async function update_plaing_timeis_kyous(): Promise<GkillError> {
    //TODO
    throw new Error('Not implemented');
}

async function request_close_dialog(): Promise<void> {
    //TODO
    throw new Error('Not implemented');
}

async function load_find_kyou_query_plaing_timeis(): Promise<void> {
    const find_plaing_timeis_query: FindTimeIsQuery = new FindTimeIsQuery()
    find_plaing_timeis_query.plaing_only = true
    const find_plaing_timeis_kyou_query: FindKyouQuery = await find_plaing_timeis_query.generate_find_kyou_query()
    find_kyou_query_plaing_timeis.value = find_plaing_timeis_kyou_query
}

load_find_kyou_query_plaing_timeis()
</script>
