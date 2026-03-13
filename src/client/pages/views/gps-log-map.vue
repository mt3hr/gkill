<template>
    <div>
        <v-sheet tile height="35" class="d-flex">
            <div class="map_date"><span>{{ start_date_str }}</span><span v-if="start_date !== end_date">～ {{
                end_date_str }}</span></div>
        </v-sheet>
        <v-sheet>
            <GoogleMap ref="gmap" :center="center" :zoom="zoom" :apiKey="google_map_api_key"
                style="width: 300px; height: 425px" class="googlemap" :key="application_config.google_map_api_key">
                <Polyline :options="polyline_options" :key="polyline_options.timestamp" />
                <Marker v-if="marker_options" :options="marker_options" :key="marker_options.timestamp" />
            </GoogleMap>
        </v-sheet>
        <v-sheet>
            <v-slider min="0" :max="time_slider_max" v-model="slider_model" :label="date_time_str" />
        </v-sheet>
    </div>
</template>
<script lang="ts" setup>
import { GoogleMap, Polyline, Marker } from 'vue3-google-map';
import type { GPSLogMapEmits } from './gps-log-map-emits'
import type { GPSLogMapProps } from './gps-log-map-props'
import { useGpsLogMap } from '@/classes/use-gps-log-map'

const props = defineProps<GPSLogMapProps>()
const emits = defineEmits<GPSLogMapEmits>()

const {
    // Template refs
    gmap,

    // State
    center,
    zoom,
    time_slider_max,
    polyline_options,
    slider_model,
    marker_options,
    google_map_api_key,

    // Computed
    start_date_str,
    end_date_str,
    date_time_str,
} = useGpsLogMap({ props, emits })
</script>
<style lang="css" scoped>
.map_date {
    font-size: 26px;
}
</style>
