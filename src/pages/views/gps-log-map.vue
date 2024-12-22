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
import moment from 'moment';
import { computed, nextTick, ref, watch, type Ref } from 'vue';
import { GoogleMap, Polyline, Marker } from 'vue3-google-map';
import type { GPSLogMapEmits } from './gps-log-map-emits'
import type { GPSLogMapProps } from './gps-log-map-props'
import { GkillAPI } from '@/classes/api/gkill-api';
import { GetGPSLogRequest } from '@/classes/api/req_res/get-gps-log-request';
import type { GPSLog } from '@/classes/datas/gps-log';
const gmap = ref<InstanceType<typeof GoogleMap> | null>(null);

const props = defineProps<GPSLogMapProps>()
const emits = defineEmits<GPSLogMapEmits>()

const center = ref({ lat: 35.6586295, lng: 139.7449018, timestamp: moment().unix() }) // mapの中心点
const zoom = ref(11) // mapのズーム
const time_slider_max = ref(86399)

watch(() => gmap.value?.ready, async () => {
    if (gmap.value && gmap.value.ready) {
        update_time_slider_max_value()
        await update_gps_log_lines()
        update_marker_by_time()
    }
})

watch(() => props.marker_time, () => {
    // start_date更新待ち
    nextTick(() => {
        slider_model.value = Math.abs(moment.duration(moment(start_date_str.value).diff(moment(props.marker_time))).asSeconds())
    })
})

const gps_logs: Ref<Array<GPSLog>> = ref([])
const polyline_options = ref({
    path: new Array<{ lat: number, lng: number }>(),
    geodesic: true,
    strokeColor: "#ff4d4d",
    strokeOpacity: 1.0,
    strokeWeight: 4,
    timestamp: moment().unix(),
}) // mapに表示するmarkerのposition
const slider_model = ref(0) // スライダーの値のモデル

const start_date_str = computed(() => moment(props.start_date).format("YYYY-MM-DD"))
const end_date_str = computed(() => moment(props.end_date).format("YYYY-MM-DD"))
const marker_options: Ref<{ position: { lat: number, lng: number }, timestamp: number } | null> = ref(null)

const google_map_api_key: Ref<string> = ref(GkillAPI.get_instance().get_google_map_api_key())

watch(() => start_date_str.value, async () => {
    update_time_slider_max_value()
    await update_gps_log_lines()
    update_marker_by_time()
})
watch(() => end_date_str.value, async () => {
    update_time_slider_max_value()
    await update_gps_log_lines()
    update_marker_by_time()
})

watch(() => slider_model.value, () => update_marker_by_time())

// datetimeが更新されたとき、sliderの値を更新し、マーカーの位置を更新する。
function update_time_slider_max_value(): void {
    let seconds = 0
    for (let date_str = start_date_str.value; !end_date_str || date_str !== end_date_str.value; date_str = moment(date_str).add(1, 'days').format("YYYY-MM-DD")) {
        seconds += 86400
    }
    seconds += 86400
    seconds--
    time_slider_max.value = seconds
}
async function update_gps_log_lines(): Promise<void> {
    const req = new GetGPSLogRequest()
    req.start_date = moment(start_date_str.value.replace("-", "/") + " 00:00:00").toDate()
    req.end_date = moment(end_date_str.value.replace("-", "/") + " 23:59:59").toDate()
    req.session_id = GkillAPI.get_instance().get_session_id()
    const res = await GkillAPI.get_instance().get_gps_log(req)
    // エラーチェック
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    res.gps_logs.sort((gps_log1, gps_log2): number => moment(gps_log1.related_time).unix() - moment(gps_log2.related_time).unix())

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
    polyline_options.value = {
        path: lines,
        geodesic: true,
        strokeColor: "#ff4d4d",
        strokeOpacity: 1.0,
        strokeWeight: 4,
        timestamp: moment().unix(),
    }
}
// timeに最も関連している地点にマーカーを立てる
function update_marker_by_time() {
    marker_options.value = null
    const datetime = moment(start_date_str.value.replace("-", "/") + " 00:00:00").add(slider_model.value, 'seconds').toDate().getTime()

    let target_gps_log: GPSLog | null = null
    for (let i = 0; i < gps_logs.value.length; i++) {
        const gps_log = gps_logs.value[i]
        if (datetime < gps_log.related_time.getTime()) {
            target_gps_log = gps_log
            break
        }
    }
    if (!target_gps_log && gps_logs.value.length !== 0) {
        target_gps_log = gps_logs.value[gps_logs.value.length - 1]
    }
    if (!target_gps_log) {
        return
    }
    marker_options.value = { position: { lat: target_gps_log.latitude.valueOf(), lng: target_gps_log.longitude.valueOf() }, timestamp: moment().unix() }
}

const date_time_str = computed(() => {
    return moment(start_date_str.value).add(slider_model.value, 'seconds').format("MM-DD HH:mm:ss")
})

async function centering(): Promise<void> {
    if (gps_logs.value.length === 0) {
        return
    }
    let minLat = 90
    let maxLat = -90
    let minLng = 180
    let maxLng = -180
    gps_logs.value.forEach(gps_log => {
        if (maxLat < gps_log.latitude.valueOf()) maxLat = gps_log.latitude.valueOf()
        if (minLat > gps_log.latitude.valueOf()) minLat = gps_log.latitude.valueOf()
        if (maxLng < gps_log.longitude.valueOf()) maxLng = gps_log.longitude.valueOf()
        if (minLng > gps_log.longitude.valueOf()) minLng = gps_log.longitude.valueOf()
    })

    const bounds = {
        north: maxLat,
        south: minLat,
        east: maxLng,
        west: minLng,
    }

    gmap.value?.map?.fitBounds(bounds)
    const msec = 100
    center.value = { lat: (minLat + maxLat) / 2, lng: (minLng + maxLng) / 2, timestamp: moment().unix() }
    await new Promise(resolve => setTimeout(resolve, msec))
    gmap.value?.map?.fitBounds(bounds)
}
// pathが更新されたとき中央寄せする
watch(() => gps_logs.value, () => centering())

update_time_slider_max_value()
</script>
<style lang="css" scoped>
.map_date {
    font-size: 26px;
}
</style>