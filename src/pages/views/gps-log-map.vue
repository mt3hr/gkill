<template>
    <div>
        <v-sheet tile height="35" class="d-flex">
            <h1><span>{{ start_date_str }}</span><span v-if="start_date !== end_date">～ {{ end_date_str }}</span></h1>
        </v-sheet>
        <v-sheet>
            <GoogleMap ref="gmap" :center="center" :zoom="zoom" :apiKey="application_config.google_map_api_key"
                style="width: 300px; height: 425px" class="googlemap">
                <Polyline v-for="line in gps_log_lines" :options="gps_log_lines" />
            </GoogleMap>
        </v-sheet>
        <v-sheet>
            <v-slider min="0" :max="time_slider_max" v-model="slider_model" :label="date_time_str"
                @click="update_marker_by_time" />
        </v-sheet>
    </div>
</template>
<script lang="ts" setup>
import moment from 'moment';
import { computed, ref, watch, type Ref } from 'vue';
import { GoogleMap, Polyline } from 'vue3-google-map';
import type { GPSLogMapEmits } from './gps-log-map-emits'
import type { GPSLogMapProps } from './gps-log-map-props'
import { GkillAPI } from '@/classes/api/gkill-api';
import { GetGPSLogRequest } from '@/classes/api/req_res/get-gps-log-request';
import type { GPSLog } from '@/classes/datas/gps-log';
const gmap = ref<InstanceType<typeof GoogleMap> | null>(null);

const props = defineProps<GPSLogMapProps>()
const emits = defineEmits<GPSLogMapEmits>()

const center = ref({ lat: 35.6586295, lng: 139.7449018 }) // mapの中心点
const zoom = ref(11) // mapのズーム
const time_slider_max = ref(86399)

const gps_logs: Ref<Array<GPSLog>> = ref([])
const gps_log_lines: Ref<google.maps.PolylineOptions> = ref({}) // mapに表示するmarkerのposition
const slider_model = ref(0) // スライダーの値のモデル

const start_date_str = computed(() => moment(props.start_date).format("YYYY-MM-DD"))
const end_date_str = computed(() => moment(props.end_date).format("YYYY-MM-DD"))

const marker_positions: Ref<Array<{ lat: number, lng: number }>> = ref([])

watch(props.start_date, () => {
    update_time_slider_max_value()
    update_gps_log_lines()
    update_marker_by_time()
})
watch(props.end_date, () => {
    update_time_slider_max_value()
    update_gps_log_lines()
    update_marker_by_time()
})

// datetimeが更新されたとき、sliderの値を更新し、マーカーの位置を更新する。
function update_time_slider_max_value(): void {
    let seconds = 86400
    for (let date_str = start_date_str.value; !end_date_str || date_str === end_date_str.value; date_str = moment(date_str).add('days', 1).format("YYYY-MM-DD")) {
        seconds += 86400
    }
    seconds--
    time_slider_max.value = seconds
}
async function update_gps_log_lines(): Promise<void> {
    const req = new GetGPSLogRequest()
    req.start_date = moment(start_date_str.value).toDate()
    req.end_date = moment(end_date_str.value).toDate()
    const res = await GkillAPI.get_instance().get_gps_log(req)
    // エラーチェック
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    res.gps_logs.sort((gps_log1, gps_log2): number => { return moment(gps_log1.related_time).unix() - moment(gps_log2.related_time).unix() })

    const lines: Array<{ lat: number, lng: number }> = []
    for (let i = 0; i < res.gps_logs.length; i++) {
        const gps_log = res.gps_logs[i]

        const line = {
            lat: gps_log.latitude.valueOf(),
            lng: gps_log.longitude.valueOf(),
        }
        lines.push(line)
    }

    gps_logs.value = res.gps_logs
    gps_log_lines.value = {
        path: lines,
        geodesic: true,
        strokeColor: "#ff4d4d",
        strokeOpacity: 1.0,
        strokeWeight: 2,
    }
}
// timeに最も関連している地点にマーカーを立てる
function update_marker_by_time() {
    marker_positions.value = []
    const datetime = moment(start_date_str.value).add('seconds', slider_model.value)

    let target_gps_log: GPSLog | null = null
    for (let i = 0; i < gps_logs.value.length; i++) {
        const gps_log = gps_logs.value[i]
        if (datetime.unix() < moment(gps_log.related_time).unix()) {
            break
        }
        target_gps_log = gps_log
    }
    if (!target_gps_log) {
        return
    }
    marker_positions.value = [{ lat: target_gps_log.latitude.valueOf(), lng: target_gps_log.longitude.valueOf() }]
}
const date_time_str = computed(() => {
    return moment(start_date_str.value).add('seconds', slider_model.value).format("MM-DD HH:mm:ss")
})

async function centering(): Promise<void> {
    let maxLat = 90
    let minLat = -90
    let maxLng = -180
    let minLng = 180
    gps_logs.value.forEach(gps_log => {
        if (maxLat < gps_log.longitude.valueOf()) maxLat = gps_log.latitude.valueOf()
        if (minLat > gps_log.latitude.valueOf()) minLat = gps_log.latitude.valueOf()
        if (maxLng < gps_log.longitude.valueOf()) maxLng = gps_log.longitude.valueOf()
        if (minLng > gps_log.longitude.valueOf()) minLng = gps_log.longitude.valueOf()
    })

    const bounds = new google.maps.LatLngBounds(
        new google.maps.LatLng(minLat, minLng),
        new google.maps.LatLng(maxLat, maxLng))
    gmap.value?.map?.fitBounds(bounds)
    const msec = 100
    await new Promise(resolve => setTimeout(resolve, msec))
    gmap.value?.map?.fitBounds(bounds)
    center.value = { lat: bounds.getCenter().lat(), lng: bounds.getCenter().lng() }
}
// pathが更新されたとき中央寄せする
watch(gps_logs, () => centering())

update_time_slider_max_value()
update_gps_log_lines()
centering()
</script>