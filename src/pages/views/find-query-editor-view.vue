<template>
    <v-card class="find_query_editor_view dummy" :class="loading_class">
        <div class="overlay_target">
            <v-overlay v-model="loading" class="align-center justify-center" persistent contained>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
        </div>
        <div v-show="inited" class="find_query_editor_view">
            <KeywordQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search="emits_current_query()" @request_update_keywords="emits_current_query()"
                @request_update_use_keyword_query="emits_current_query()"
                @request_clear_keyword_query="emits_cleard_keyword_query()"
                :inited="inited_keyword_query_for_query_sidebar" @inited="inited_keyword_query_for_query_sidebar = true"
                ref="keyword_query" />
            <div> <v-divider /> </div>
            <RepQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_checked_reps="emits_current_query()" @request_clear_rep_query="emits_cleard_rep_query()"
                ref="rep_query" :inited="inited_rep_query_for_query_sidebar"
                @inited="inited_rep_query_for_query_sidebar = true" />
            <div> <v-divider /> </div>
            <TagQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_tags="emits_current_query()"
                @request_update_checked_tags="emits_current_query()" @request_clear_tag_query="emits_cleard_tag_query()"
                ref="tag_query" :inited="inited_tag_query_for_query_sidebar"
                @inited="inited_tag_query_for_query_sidebar = true" />
            <div> <v-divider /> </div>
            <TimeIsQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_and_search_timeis_tags="emits_current_query()"
                @request_update_and_search_timeis_word="emits_current_query()"
                @request_update_checked_timeis_tags="emits_current_query()"
                :inited="inited_timeis_query_for_query_sidebar" @inited="inited_timeis_query_for_query_sidebar = true"
                @request_update_timeis_keywords="emits_current_query()"
                @request_update_use_timeis_query="emits_current_query()"
                @request_clear_timeis_query="emits_cleard_timeis_query()" ref="timeis_query" />
            <div> <v-divider /> </div>
            <MapQuery :application_config="application_config" :gkill_api="gkill_api" :find_kyou_query="query"
                @request_update_area="emits_current_query()" @request_update_use_map_query="emits_current_query()"
                @request_clear_map_query="emits_cleard_map_query()" :inited="inited_map_query_for_query_sidebar"
                @inited="inited_map_query_for_query_sidebar = true" ref="map_query" />
        </div>
        <v-card-action v-if="!loading">
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary"
                        @click="() => { emits('requested_apply', generate_query(gkill_api.generate_uuid())); emits('requested_close_dialog') }">{{
                            i18n.global.t("SAVE_TITLE") }}</v-btn>
                </v-col>
                <v-spacer class="pa-0 ma-0" />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits_default_query()">{{
                        i18n.global.t("CLEAR_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import KeywordQuery from './keyword-query.vue'
import MapQuery from './map-query.vue'
import RepQuery from './rep-query.vue'
import TagQuery from './tag-query.vue'
import TimeIsQuery from './time-is-query.vue'
import { computed, nextTick, onMounted, type Ref, ref, watch } from 'vue'
import { deepEquals } from '@/classes/deep-equals'
import moment from 'moment'
import { CheckState } from './check-state'
import type { FindQueryEditorViewProps } from './find-query-editor-view-props'
import type { FindQueryEditorViewEmits } from './find-query-editor-view-emits'

const keyword_query = ref<InstanceType<typeof KeywordQuery> | null>(null);
const timeis_query = ref<InstanceType<typeof TimeIsQuery> | null>(null);
const rep_query = ref<InstanceType<typeof RepQuery> | null>(null);
const tag_query = ref<InstanceType<typeof TagQuery> | null>(null);
const map_query = ref<InstanceType<typeof MapQuery> | null>(null);

const props = defineProps<FindQueryEditorViewProps>()
const emits = defineEmits<FindQueryEditorViewEmits>()
defineExpose({ generate_query, get_default_query })

const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

const is_mounted = ref(false)
onMounted(() => is_mounted.value = true)

const loading: Ref<boolean> = ref(true)
const loading_class = computed(() => loading.value ? "loading_find_query_editor_view" : "")
const inited = computed(() => {
    if (!is_mounted.value) {
        return false
    }
    return inited_keyword_query_for_query_sidebar.value &&
        inited_rep_query_for_query_sidebar.value &&
        inited_tag_query_for_query_sidebar.value &&
        inited_timeis_query_for_query_sidebar.value &&
        inited_map_query_for_query_sidebar.value
})

watch(() => inited.value, (new_value: boolean, old_value: boolean) => {
    if (old_value !== new_value && new_value) {
        default_query.value = generate_query().clone()
        default_query.value.query_id = props.gkill_api.generate_uuid()
        nextTick(() => { emits('inited') })
        nextTick(() => {
            if (props.find_kyou_query.query_id === "") {
                query.value = default_query.value
            } else {
                query.value = props.find_kyou_query
            }
        })
        nextTick(() => loading.value = false)
    }
})

const inited_keyword_query_for_query_sidebar = ref(true)
const inited_timeis_query_for_query_sidebar = ref(false)
const inited_rep_query_for_query_sidebar = ref(false)
const inited_tag_query_for_query_sidebar = ref(false)
const inited_map_query_for_query_sidebar = ref(false)

watch(() => props.find_kyou_query, (new_value: FindKyouQuery, old_value: FindKyouQuery) => {
    if (deepEquals(new_value, old_value)) {
        return
    }
    query.value = new_value
})

function get_default_query(): FindKyouQuery {
    const query = default_query.value?.clone()
    query.query_id = props.gkill_api.generate_uuid()
    if (props.application_config.rykv_default_period !== -1) {
        query.use_calendar = true
        query.calendar_start_date = moment(moment().add(-props.application_config.rykv_default_period, "days").format("YYYY-MM-DD 00:00:00 ZZ")).toDate()
        query.calendar_end_date = moment(moment().format("YYYY-MM-DD 00:00:00 ZZ")).add(1, "days").add(-1, "milliseconds").toDate()
    }
    return query
}

function emits_current_query(): void {
    const currentt_query = generate_query(query.value.query_id)
}

function generate_query(query_id?: string): FindKyouQuery {
    const find_query = new FindKyouQuery()
    if (query_id) {
        find_query.query_id = query_id
    }

    find_query.is_focus_kyou_in_list_view = props.find_kyou_query! ? props.find_kyou_query.is_focus_kyou_in_list_view : false
    find_query.is_image_only = props.find_kyou_query ? props.find_kyou_query.is_image_only : false

    if (keyword_query.value) {
        find_query.use_words = keyword_query.value.get_use_words()
        find_query.words_and = keyword_query.value.get_use_word_and_search()
        find_query.keywords = keyword_query.value.get_keywords().concat()
    }

    if (timeis_query.value) {
        find_query.use_timeis = timeis_query.value.get_use_timeis()
        find_query.timeis_keywords = timeis_query.value.get_timeis_keywords().concat()
        find_query.timeis_words_and = timeis_query.value.get_use_and_search_timeis_words()
        find_query.use_timeis_tags = timeis_query.value.get_use_timeis_tags()
        find_query.timeis_tags = timeis_query.value.get_timeis_tags().concat()
        find_query.timeis_tags_and = timeis_query.value.get_use_and_search_timeis_tags()
    }

    if (rep_query.value) {
        const reps = rep_query.value.get_checked_reps()?.concat()
        const devices = rep_query.value.get_checked_devices()?.concat()
        const rep_types = rep_query.value.get_checked_rep_types()?.concat()
        if (reps) {
            find_query.reps = reps
        }
        if (devices) {
            find_query.devices_in_sidebar = devices
        }
        if (rep_types) {
            find_query.rep_types_in_sidebar = rep_types
        }
    }

    if (tag_query.value) {
        const tags = tag_query.value.get_tags()?.concat()
        if (tags) {
            find_query.tags = tags
        }
        find_query.tags_and = tag_query.value.get_is_and_search()
    }

    if (map_query.value) {
        find_query.use_map = map_query.value.get_use_map()
        find_query.map_latitude = map_query.value.get_latitude()
        find_query.map_longitude = map_query.value.get_longitude()
        find_query.map_radius = map_query.value.get_radius()
        find_query.is_enable_map_circle_in_sidebar = map_query.value.get_is_enable_circle()
    }

    return find_query
}

function emits_cleard_keyword_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.use_words = get_default_query().use_words
    find_query.keywords = get_default_query().keywords.concat()
    find_query.words_and = get_default_query().words_and
    query.value = find_query
}

function emits_cleard_timeis_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.use_timeis = get_default_query().use_timeis
    find_query.use_timeis_tags = get_default_query().use_timeis_tags
    find_query.timeis_keywords = get_default_query().timeis_keywords.concat()
    find_query.timeis_words_and = get_default_query().timeis_words_and
    find_query.use_timeis_tags = get_default_query().use_timeis_tags
    find_query.timeis_tags = get_default_query().timeis_tags.concat()
    find_query.timeis_tags_and = get_default_query().timeis_tags_and
    query.value = find_query
    timeis_query.value?.update_check(find_query.tags, CheckState.checked, true, true)
}

function emits_cleard_rep_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.reps = get_default_query().reps.concat()
    find_query.devices_in_sidebar = get_default_query().devices_in_sidebar.concat()
    find_query.rep_types_in_sidebar = get_default_query().rep_types_in_sidebar.concat()
    query.value = find_query
}

function emits_cleard_tag_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.tags = get_default_query().tags.concat()
    find_query.tags_and = get_default_query().tags_and
    query.value = find_query
    tag_query.value?.update_check(find_query.tags, CheckState.checked, true, true)
}

function emits_cleard_map_query(): void {
    const find_query = generate_query()
    find_query.query_id = props.gkill_api.generate_uuid()
    find_query.use_map = get_default_query().use_map
    find_query.map_latitude = get_default_query().map_latitude
    find_query.map_longitude = get_default_query().map_longitude
    find_query.is_enable_map_circle_in_sidebar = get_default_query().is_enable_map_circle_in_sidebar
    find_query.map_radius = get_default_query().map_radius
    query.value = find_query
}

async function emits_default_query(): Promise<void> {
    const find_query = get_default_query().clone()
    find_query.query_id = props.gkill_api.generate_uuid()
    await tag_query.value?.update_check(find_query.tags, CheckState.checked, true, true)
    await timeis_query.value?.update_check(find_query.tags, CheckState.checked, true, true)
    query.value = find_query
}
</script>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    display: flex;
    flex-direction: column;
    align-items: center;
    min-height: 40vh;
    min-width: 40vw;
}

.find_query_editor_view.dummy {
    position: relative;
    min-height: 40vh;
    min-width: 40vw;
}

.loading_find_query_editor_view {
    overflow-y: hidden !important;
    min-width: calc(40vw + 8px);
}
</style>