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
        <GoogleMap ref="gmap" :center="center" :zoom="zoom" :apiKey="google_map_api_key" @click="handle_map_click"
            style="width: 100%; height: 400px" class="googlemap search_google_map"
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
import { ref } from 'vue'
import { Circle, GoogleMap } from 'vue3-google-map'
import type { MapQueryEmits } from './map-query-emits'
import type { MapQueryProps } from './map-query-props'
import { useMapQuery } from '@/classes/use-map-query'

const props = defineProps<MapQueryProps>()
const emits = defineEmits<MapQueryEmits>()

const gmap = ref<InstanceType<typeof GoogleMap> | null>(null)

const {
    query,
    google_map_api_key,
    radius,
    zoom,
    center,
    circle,
    handle_map_click,
    get_use_map,
    get_latitude,
    get_longitude,
    get_radius,
    get_is_enable_circle,
} = useMapQuery({ props, emits })

defineExpose({ get_use_map, get_latitude, get_longitude, get_radius, get_is_enable_circle })
</script>
