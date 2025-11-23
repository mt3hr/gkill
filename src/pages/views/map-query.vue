<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <v-checkbox v-model="query.use_map" @change=" emits('request_update_use_map_query', query.use_map)"
                :label="i18n.global.t('MAP_QUERY_TITLE')" hide-details class="pb-0 mb-0" />
        </v-col>
        <v-spacer class="pa-0 ma-0" />
        <v-col cols="auto" class="pb-0 mb-0 pr-0">
            <v-btn dark color="secondary" @click="emits('request_clear_map_query')">{{ i18n.global.t("CLEAR_TITLE")
                }}</v-btn>
        </v-col>
    </v-row>
    <v-sheet v-show="query.use_map">
        <GoogleMap ref="gmap" :center="center" :zoom="zoom" :apiKey="google_map_api_key" @click="($event) => {
            is_enable_circle = true;
            latitude = $event.latLng.lat();
            longitude = $event.latLng.lng();
            emits('request_update_area', $event.latLng.lat(), $event.latLng.lng(), radius)
        }" style="width: 100%; height: 400px" class="googlemap search_google_map"
            :key="application_config.google_map_api_key">
            <Circle :options="circle"
                :key="(circle.center?.lat.toString().concat(circle.center?.lng.toString()).concat(radius.toString()))" />
        </GoogleMap>
    </v-sheet>
    <v-sheet v-show="query.use_map">
        <v-slider min="0" max="10000" v-model="radius" :label="i18n.global.t('RANGE_TITLE')" />
    </v-sheet>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'

import { Circle, GoogleMap } from 'vue3-google-map';
import type { MapQueryEmits } from './map-query-emits'
import type { MapQueryProps } from './map-query-props'
import { computed, ref, watch, type Ref } from 'vue';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';

const props = defineProps<MapQueryProps>()
const emits = defineEmits<MapQueryEmits>()
defineExpose({ get_use_map, get_latitude, get_longitude, get_radius, get_is_enable_circle })

const gmap = ref<InstanceType<typeof GoogleMap> | null>(null);

const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

const google_map_api_key: Ref<string> = ref(props.gkill_api.get_google_map_api_key())

const latitude: Ref<number> = ref(35.6586295)
const longitude: Ref<number> = ref(139.7449018)
const radius: Ref<number> = ref(500)

const zoom = ref(11) // mapのズーム
const is_enable_circle = ref(query.value.is_enable_map_circle_in_sidebar)

const center = ref({ lat: 35.6586295, lng: 139.7449018 })
const circle = computed(() => {
    return {
        visible: is_enable_circle.value,
        center: { lat: latitude.value, lng: longitude.value },
        radius: radius.value,
        strokeColor: 'black',
        strokeOpacity: 1,
        strokeWeight: 2,
    }
})

watch(() => props.find_kyou_query, () => {
    if (JSON.stringify(query.value) === JSON.stringify(props.find_kyou_query)) {
        return
    }
    query.value = props.find_kyou_query.clone()
    latitude.value = props.find_kyou_query.map_latitude.valueOf()
    longitude.value = props.find_kyou_query.map_longitude.valueOf()
    is_enable_circle.value = props.find_kyou_query.is_enable_map_circle_in_sidebar
    radius.value = props.find_kyou_query.map_radius.valueOf()
    emits('request_update_area', latitude.value, longitude.value, radius.value)
})

watch(() => props.application_config, async () => {
    emits('inited')
})

watch(() => radius.value, () => {
    emits('request_update_area', latitude.value, longitude.value, radius.value)
})

watch(() => props.application_config, () => {
    google_map_api_key.value = props.application_config.google_map_api_key
})

function get_use_map(): boolean {
    return query.value.use_map
}
function get_latitude(): number {
    return latitude.value
}
function get_longitude(): number {
    return longitude.value
}
function get_radius(): number {
    return radius.value
}
function get_is_enable_circle(): boolean {
    return is_enable_circle.value
}
</script>
