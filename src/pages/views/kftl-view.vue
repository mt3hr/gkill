<template>
    <KFTLLineLabel v-for="line_label_data, index in line_label_datas" :application_config="application_config"
        :style="line_label_styles[index]" :gkill_api="gkill_api" :line_label_data="line_label_data" />
    <KFTLTemplateDialog :application_config="application_config" :gkill_api="gkill_api"
        :templates="[application_config.parsed_kftl_template]" />
</template>

<script setup lang="ts">
import { computed, ref, type Ref } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { FindTimeIsQuery } from '@/classes/api/find_query/find-time-is-query'
import { LineLabelData } from '@/classes/kftl/line-label-data'

import type { KFTLProps } from './kftl-props'
import type { KFTLViewEmits } from './kftl-view-emits'

import KyouListView from './kyou-list-view.vue'
import KFTLLineLabel from './kftl-line-label.vue'
import KFTLTemplateDialog from '../dialogs/kftl-template-dialog.vue'
import type { Kyou } from '@/classes/datas/kyou'

const text_area_content: Ref<string> = ref("")
const text_area_width: Ref<Number> = ref(0)
const text_area_width_px = computed(() => text_area_width.value.toString().concat("px"))
const text_area_height: Ref<Number> = ref(0)
const text_area_height_px = computed(() => text_area_height.value.toString().concat("px"))
const line_label_width: Ref<Number> = ref(0)
const line_label_width_px = computed(() => line_label_width.value.toString().concat("px"))
const line_label_height: Ref<Number> = ref(0)
const line_label_height_px = computed(() => line_label_height.value.toString().concat("px"))

const line_label_datas: Ref<Array<LineLabelData>> = ref(new Array<LineLabelData>())
const line_label_styles: Ref<Array<string>> = ref(new Array<string>())
const invalid_line_numbers: Ref<Array<Number>> = ref(new Array<Number>())
const is_requested_submit: Ref<boolean> = ref(false)
const find_kyou_query_plaing_timeis: Ref<FindKyouQuery> = ref(new FindKyouQuery())
const plaing_timeis_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())

const last_added_tag: Ref<string> = ref("")

const props = defineProps<KFTLProps>()
const emits = defineEmits<KFTLViewEmits>()

async function restore_content_from_localstorage(): Promise<void> {
    //TODO
    throw new Error('Not implemented')
}

async function save_content_to_localstorage(): Promise<void> {
    //TODO
    throw new Error('Not implemented')
}

async function update_line_labels(): Promise<void> {
    //TODO
    throw new Error('Not implemented')
}

async function is_invalid_line(line_index: Number): Promise<boolean> {
    //TODO
    throw new Error('Not implemented')
}

async function submit(): Promise<Array<GkillError>> {
    //TODO
    throw new Error('Not implemented')
}

async function clear(): Promise<void> {
    //TODO
    throw new Error('Not implemented')
}

async function show_kftl_template_dialog(): Promise<void> {
    //TODO
    throw new Error('Not implemented')
}

async function resize(): Promise<void> {
    //TODO
    throw new Error('Not implemented')
}

async function apply_application_config(): Promise<Array<GkillError>> {
    //TODO
    throw new Error('Not implemented')
}

async function update_plaing_timeis_kyous(): Promise<GkillError> {
    //TODO
    throw new Error('Not implemented')
}

async function request_close_dialog(): Promise<void> {
    //TODO
    throw new Error('Not implemented')
}

async function load_find_kyou_query_plaing_timeis(): Promise<void> {
    const find_plaing_timeis_query: FindTimeIsQuery = new FindTimeIsQuery()
    find_plaing_timeis_query.plaing_only = true
    const find_plaing_timeis_kyou_query: FindKyouQuery = await find_plaing_timeis_query.generate_find_kyou_query()
    find_kyou_query_plaing_timeis.value = find_plaing_timeis_kyou_query
}

async function reload_plaing_timeis(kyou: Kyou): Promise<void> {
    let index = -1
    for (let i = 0; i < plaing_timeis_kyous.value.length; i++) {
        const kyou_in_list = plaing_timeis_kyous.value[i]
        if (kyou.id === kyou_in_list.id) {
            index = i
            await kyou_in_list.reload()
            break
        }
    }
    if (index !== -1) {
        plaing_timeis_kyous.value.splice(index, 1, plaing_timeis_kyous.value[index])
    }
}

load_find_kyou_query_plaing_timeis()
</script>
