<template>
    <v-virtual-scroll v-if="!query.is_image_only" class="kyou_list_view" :items="matched_kyous"
        :item-height="kyou_height_px" :height="list_height" :width="width">
        <template v-slot:default="{ item }">
            <KyouView class="kyou_in_list" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :is_image_view="false" :kyou="item" :last_added_tag="last_added_tag"
                :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true"
                :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                :show_timeis_plaing_end_button="true" @received_errors="(errors) => emits('received_errors', errors)"
                :height="kyou_height.valueOf()" :width="width.valueOf()"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </template>
    </v-virtual-scroll>
    <v-virtual-scroll v-if="query.is_image_only" class="kyou_list_view_image" :items="match_kyous_for_image"
        :item-height="kyou_height_px" :height="list_height"
        :width="200 * application_config.rykv_image_list_column_number.valueOf()">
        <template v-slot:default="{ item }">
            <table>
                <tr>
                    <td v-for="kyou in item">
                        <KyouView class="kyou_image_in_list" :application_config="application_config"
                            :gkill_api="gkill_api" :highlight_targets="[]" :is_image_view="false" :kyou="kyou"
                            :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                            :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                            :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                            :show_timeis_plaing_end_button="true" :height="'100%'" :width="'100%'"
                            @received_errors="(errors) => emits('received_errors', errors)"
                            @received_messages="(messages) => emits('received_messages', messages)"
                            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                            @requested_reload_list="() => { }"
                            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
                    </td>
                </tr>
            </table>
        </template>
    </v-virtual-scroll>
</template>
<script setup lang="ts">
import type { KyouListViewEmits } from './kyou-list-view-emits'
import type { KyouListViewProps } from './kyou-list-view-props'
import { GkillError } from '@/classes/api/gkill-error'
import { Kyou } from '@/classes/datas/kyou'
import { computed, type Ref, ref, watch } from 'vue'
import KyouView from './kyou-view.vue'

const props = defineProps<KyouListViewProps>()
const emits = defineEmits<KyouListViewEmits>()
// const cloned_find_query: Ref<FindKyouQuery> = ref(await props.query.clone())
const match_kyous: Ref<Array<Kyou>> = ref(props.matched_kyous ? props.matched_kyous.concat() : new Array<Kyou>())
const match_kyous_for_image: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
const is_loading: Ref<boolean> = ref(false)
const scroll_distance_from_top_px: Ref<Number> = ref(0)
const checked_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const kyou_height_px = computed(() => props.kyou_height ? props.kyou_height.toString().concat("px") : "0px")

watch(() => match_kyous, () => {
    if (props.query.is_image_only) {
        update_match_kyous_for_image()
    }
})
watch(() => props.query, () => {
    if (props.query.is_image_only) {
        update_match_kyous_for_image()
    }
})

async function update_match_kyous_for_image(): Promise<void> {
    match_kyous_for_image.value = []
    const match_kyous_for_image_result = new Array<Array<Kyou>>()
    for (let i = 0; i < match_kyous.value.length;) {
        const kyou_row_list = new Array<Kyou>()
        for (let j = 0; j < props.application_config.rykv_image_list_column_number.valueOf(); j++) {
            const kyou = match_kyous.value[i]
            kyou_row_list.push(kyou)
            i++
        }
        match_kyous_for_image_result.push(kyou_row_list)
    }
    match_kyous_for_image.value = match_kyous_for_image_result
}

async function reload(): Promise<Array<GkillError>> {
    throw new Error('Not implemented')
}
async function scroll_to_kyou(kyou: Kyou): Promise<boolean> {
    throw new Error('Not implemented')
}
async function scroll_to_time(time: Date): Promise<boolean> {
    throw new Error('Not implemented')
}
</script>

<style lang="css" scoped>
.kyou_in_list {
    overflow-y: hidden !important;
    height: v-bind(kyou_height_px) !important;
    min-height: v-bind(kyou_height_px) !important;
    max-height: v-bind(kyou_height_px) !important;
    border-top: 1px solid silver;
}

.kyou_image_in_list {
    height: 200px;
    width: 200px;
}
</style>