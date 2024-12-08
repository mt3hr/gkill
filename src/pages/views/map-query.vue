<template>
    <v-row>
        <v-col cols="auto" class="pb-0 mb-0">
            <v-checkbox v-model="query.use_map" @change=" emits('request_update_use_map_query', query.use_map)"
                label="場所" hide-details />
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pb-0 mb-0">
            <v-btn @click="circles = []; emits('request_clear_map_query')">クリア</v-btn>
        </v-col>
    </v-row>
    <v-sheet v-show="query.use_map">
        <GoogleMap ref="gmap" :center="center" :zoom="zoom" :apiKey="application_config.google_map_api_key"
            @click="($event) => { update_circles(); center.lat = $event.latLng.lat(); center.lng = $event.latLng.lng(); is_enable_circle = true; emits('request update_area', center.lat, center.lng, query.map_radius); is_enable_circle = true; }"
            style="width: 100%; height: 400px" class="googlemap search_google_map">
            <Circle v-for="opt in circles" :options="opt"
                :key="(opt.center?.lat.toString().concat(opt.center?.lng.toString()).concat(query.map_radius.toString()))" />
        </GoogleMap>
    </v-sheet>
    <v-sheet v-show="query.use_map">
        <v-slider min="0" max="5000" v-model="query.map_radius" :label="'範囲'"
            @click="update_circles(); emits('request update_area', center.lat, center.lng, query.map_radius)" />
    </v-sheet>
</template>
<script lang="ts" setup>

import { Circle, GoogleMap } from 'vue3-google-map';
import type { MapQueryEmits } from './map-query-emits'
import type { MapQueryProps } from './map-query-props'
import { computed, ref, watch, type Ref } from 'vue';
import type { CircleOptions } from '@/classes/datas/circle-options';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';

const props = defineProps<MapQueryProps>()
const emits = defineEmits<MapQueryEmits>()
defineExpose({ get_use_map, get_latitude, get_longitude, get_radius })

const gmap = ref<InstanceType<typeof GoogleMap> | null>(null);

const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

const center = ref({ lat: 35.6586295, lng: 139.7449018 }) // mapの中心点
const zoom = ref(11) // mapのズーム
const is_enable_circle = ref(false)
const circles: Ref<Array<CircleOptions>> = ref(new Array<CircleOptions>())

watch(() => props.find_kyou_query, () => {
    query.value = props.find_kyou_query.clone()
    center.value.lat = props.find_kyou_query.map_latitude.valueOf()
    center.value.lng = props.find_kyou_query.map_longitude.valueOf()
})

function update_circles(): void {
    circles.value = []
    circles.value.push(
        {
            visible: is_enable_circle.value,
            center: center.value,
            radius: query.value.map_radius.valueOf(),
            strokeColor: 'black',
            strokeOpacity: 0.7,
            strokeWeight: 2
        }
    )
}

function get_use_map(): boolean {
    return query.value.use_map
}
function get_latitude(): number {
    return center.value.lat
}
function get_longitude(): number {
    return center.value.lng
}
function get_radius(): number {
    return query.value.map_radius.valueOf()
}
</script>
