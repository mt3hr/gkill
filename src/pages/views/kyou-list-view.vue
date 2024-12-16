<template>
    <v-card :ripple="false">
        <v-card :ripple="false">
            <v-overlay v-model="is_loading" class="align-center justify-center" contained persistent>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
            <v-virtual-scroll v-if="!query.is_image_only" class="kyou_list_view" :items="matched_kyous"
                :item-height="kyou_height_px" :height="list_height.valueOf() - footer_height.valueOf()" :width="width"
                ref="kyou_list_view">
                <template v-slot:default="{ item }">
                    <KyouView class="kyou_in_list" :application_config="application_config" :gkill_api="gkill_api"
                        :key="item.id" :highlight_targets="[]" :is_image_view="false" :kyou="item"
                        :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
                        :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                        :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                        :show_timeis_plaing_end_button="true" :width="width.valueOf()" :is_readonly_mi_check="true"
                        @received_errors="(errors) => emits('received_errors', errors)" :height="kyou_height.valueOf()"
                        @clicked_kyou="(kyou) => emits('clicked_kyou', kyou)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                        @requested_reload_list="() => { }"
                        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
                </template>
            </v-virtual-scroll>
            <v-virtual-scroll v-if="query.is_image_only" class="kyou_list_view_image" :items="match_kyous_for_image"
                :item-height="kyou_height_px" :height="list_height.valueOf() - footer_height.valueOf()"
                :width="200 * application_config.rykv_image_list_column_number.valueOf()">
                <template v-slot:default="{ item }">
                    <table>
                        <tr>
                            <td v-for="kyou in item">
                                <KyouView class="kyou_image_in_list" :application_config="application_config"
                                    :key="kyou.id" :gkill_api="gkill_api" :highlight_targets="[]" :is_image_view="false"
                                    :kyou="kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
                                    :show_content_only="true" :show_mi_create_time="true"
                                    :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                                    :show_mi_limit_time="true" :show_timeis_plaing_end_button="true" :height="'100%'"
                                    :width="'100%'" :is_readonly_mi_check="true"
                                    @received_errors="(errors) => emits('received_errors', errors)"
                                    @clicked_kyou="(kyou) => emits('clicked_kyou', kyou)"
                                    @received_messages="(messages) => emits('received_messages', messages)"
                                    @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                                    @requested_reload_list="() => { }"
                                    @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
                                    ref="kyou_list_image_view" />
                            </td>
                        </tr>
                    </table>
                </template>
            </v-virtual-scroll>
        </v-card>
        <v-card :class="footer_class" :ripple="false">
            <v-row no-gutters>
                <v-col v-if="matched_kyous && matched_kyous.length" cols="auto" class="py-2">
                    {{ matched_kyous.length }}件のアイテム
                </v-col>
                <v-spacer />

                <v-col cols="auto" class="pa-0">
                    <v-btn class="rounded-sm mx-auto" icon @click="emits('requested_search')" variant="text">
                        <v-icon>mdi-reload</v-icon>
                    </v-btn>
                </v-col>

                <v-col cols="auto" class="pa-0">
                    <v-btn class="rounded-sm mx-auto" icon
                        @click="emits('requested_change_is_image_only_view', !query.is_image_only)" variant="text">
                        <v-icon v-show="!query.is_image_only">mdi-file-document-outline</v-icon>
                        <v-icon v-show="query.is_image_only">mdi-image</v-icon>
                    </v-btn>
                </v-col>

                <v-col cols="auto" class="pa-0">
                    <v-btn class="rounded-sm mx-auto" icon variant="text"
                        @click="emits('requested_change_focus_kyou', !query.is_focus_kyou)">
                        <v-icon v-show="!query.is_focus_kyou">mdi-arrow-down</v-icon>
                        <v-icon v-show="query.is_focus_kyou">mdi-arrow-right</v-icon>
                    </v-btn>
                </v-col>

                <v-col cols="auto" class="pa-0">
                    <v-btn class="rounded-sm mx-auto" icon @click="emits('requested_close_column')" variant="text">
                        <v-icon>mdi-close</v-icon>
                    </v-btn>
                </v-col>
            </v-row>
        </v-card>
    </v-card>
</template>
<script setup lang="ts">
import type { KyouListViewEmits } from './kyou-list-view-emits'
import type { KyouListViewProps } from './kyou-list-view-props'
import { GkillError } from '@/classes/api/gkill-error'
import { Kyou } from '@/classes/datas/kyou'
import { computed, type Ref, ref, watch } from 'vue'
import KyouView from './kyou-view.vue'
import type { VVirtualScroll } from 'vuetify/components'
const kyou_list_view = ref<InstanceType<typeof VVirtualScroll> | null>(null);
const kyou_list_image_view = ref<InstanceType<typeof VVirtualScroll> | null>(null);

const props = defineProps<KyouListViewProps>()
const emits = defineEmits<KyouListViewEmits>()
defineExpose({ scroll_to_kyou, scroll_to_time, set_loading })

const match_kyous: Ref<Array<Kyou>> = ref(props.matched_kyous ? props.matched_kyous.concat() : new Array<Kyou>())
const match_kyous_for_image: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
const is_loading: Ref<boolean> = ref(false)
const scroll_distance_from_top_px: Ref<Number> = ref(0)
const checked_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const kyou_height_px = computed(() => props.kyou_height ? props.kyou_height.toString().concat("px") : "0px")

const footer_height: Ref<number> = ref(48)
const footer_height_px = computed(() => footer_height.value.toString().concat("px"))

watch(() => props.query, () => {
    if (props.query.is_image_only) {
        update_match_kyous_for_image()
    }
})

watch(() => props.matched_kyous, () => {
    if (props.query.is_image_only) {
        update_match_kyous_for_image()
    }
})

const footer_class = computed(() => {
    return props.is_focused_list ? 'focused_list' : ''
})

async function update_match_kyous_for_image(): Promise<void> {
    match_kyous_for_image.value = []
    const match_kyous_for_image_result = new Array<Array<Kyou>>()
    for (let i = 0; i < props.matched_kyous.length;) {
        const kyou_row_list = new Array<Kyou>()
        for (let j = 0; j < props.application_config.rykv_image_list_column_number.valueOf(); j++) {
            const kyou = props.matched_kyous[i]
            kyou_row_list.push(kyou)
            i++
        }
        match_kyous_for_image_result.push(kyou_row_list)
    }
    match_kyous_for_image.value.splice(0, match_kyous_for_image.value.length - 1)
    match_kyous_for_image.value.push(...match_kyous_for_image_result)
}

async function reload(): Promise<Array<GkillError>> {
    kyou_list_view.value?.$forceUpdate()
    kyou_list_image_view.value?.$forceUpdate()
    return []
}
async function scroll_to_kyou(kyou: Kyou): Promise<boolean> {
    let index = -1;
    for (let i = 0; i < props.matched_kyous.length; i++) {
        const kyou_in_list = props.matched_kyous[i]
        if (kyou_in_list.id === kyou.id) {
            index = i
            break
        }
    }

    if (index === -1) {
        return false
    }
    kyou_list_view.value?.scrollToIndex(index)
    kyou_list_image_view.value?.scrollToIndex(index / props.application_config.rykv_image_list_column_number.valueOf())
    return true
}
async function scroll_to_time(time: Date): Promise<boolean> {
    let index = -1;
    for (let i = 0; i < props.matched_kyous.length; i++) {
        const kyou = props.matched_kyous[i]
        if (kyou.related_time.getTime() <= time.getTime()) {
            index = i
            break
        }
    }

    if (index === -1) {
        return false
    }
    kyou_list_view.value?.scrollToIndex(index)
    kyou_list_image_view.value?.scrollToIndex(index / props.application_config.rykv_image_list_column_number.valueOf())
    return true
}

function set_loading(loading: boolean): void {
    is_loading.value = loading
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

.focused_list>* {
    background-color: silver;
}
</style>