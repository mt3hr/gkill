<template>
    <div class="gps_log_map_wrap">
        <v-sheet tile height="35" class="d-flex">
            <v-menu v-model="is_show_date_picker" :close-on-content-click="false" location="bottom start"
                :z-index="3000">
                <template v-slot:activator="{ props: dateMenuProps }">
                    <v-btn variant="text" class="map_date map_date_button text-none" v-bind="dateMenuProps">
                        <span>{{ start_date_str }}</span><span v-if="start_date_str !== end_date_str">～ {{
                            end_date_str }}</span>
                    </v-btn>
                </template>
                <v-date-picker v-model="date_picker_model" />
            </v-menu>
        </v-sheet>
        <div class="map_container">
            <GoogleMap ref="gmap" :center="center" :zoom="zoom" :apiKey="google_map_api_key"
                class="googlemap"
                :key="application_config.google_map_api_key + (application_config.use_dark_theme ? '_dark' : '_light')"
                :colorScheme="application_config.use_dark_theme ? 'DARK' : 'LIGHT'"
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
    is_show_date_picker,

    // Computed
    start_date_str,
    end_date_str,
    date_time_str,
    date_picker_model,

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

/* 日付表示は日付ピッカーを開くボタン。v-btn の既定の文字サイズ・高さに負けないよう2クラスで指定する */
.v-btn.map_date_button {
    font-size: 26px;
    height: 35px;
    letter-spacing: normal;
    padding: 0 8px;
}
</style>
