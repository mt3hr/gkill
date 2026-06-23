<template>
    <div class="gps_log_map_wrap">
        <v-sheet tile height="35" class="d-flex">
            <div class="map_date"><span>{{ start_date_str }}</span><span v-if="start_date !== end_date">～ {{
                end_date_str }}</span></div>
        </v-sheet>
        <div class="map_container">
            <GoogleMap ref="gmap" :center="center" :zoom="zoom" :apiKey="google_map_api_key"
                class="googlemap" :key="application_config.google_map_api_key"
                gestureHandling="cooperative">
                <Polyline :options="polyline_options" :key="polyline_options.timestamp" />
                <Marker v-if="marker_options" :options="marker_options" :key="marker_options.timestamp" />
            </GoogleMap>
        </div>
        <v-sheet>
            <v-slider min="0" hide-details :max="time_slider_max" v-model="slider_model" :label="date_time_str" />
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

    centering,
} = useGpsLogMap({ props, emits })

defineExpose({ centering })
</script>
<style lang="css" scoped>
.gps_log_map_wrap {
    display: flex;
    flex-direction: column;
    height: v-bind('app_content_height.toString().concat("px")');
    width: 400px;
}

.map_container {
    flex: 1;
    overflow: hidden;
    min-height: 0;
}

.googlemap {
    width: 100%;
    height: 100%;
}

.map_date {
    font-size: 26px;
}
</style>
