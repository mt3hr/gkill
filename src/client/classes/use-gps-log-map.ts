import moment from 'moment'
import { computed, nextTick, ref, watch, type Ref } from 'vue'
import type { GoogleMap } from 'vue3-google-map'
import type { GPSLogMapEmits } from '@/pages/views/gps-log-map-emits'
import type { GPSLogMapProps } from '@/pages/views/gps-log-map-props'
import { GetGPSLogRequest } from '@/classes/api/req_res/get-gps-log-request'
import type { GPSLog } from '@/classes/datas/gps-log'

export function useGpsLogMap(options: {
    props: GPSLogMapProps,
    emits: GPSLogMapEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const gmap = ref<InstanceType<typeof GoogleMap> | null>(null)

    // ── State refs ──
    const center = ref({ lat: 35.6586295, lng: 139.7449018, timestamp: moment().unix() }) // mapの中心点
    const zoom = ref(11) // mapのズーム
    const time_slider_max = ref(86399)
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
    const marker_options: Ref<{ position: { lat: number, lng: number }, timestamp: number } | null> = ref(null)
    const google_map_api_key: Ref<string> = ref(props.gkill_api.get_google_map_api_key())

    // ── Computed ──
    const start_date_str = computed(() => moment(props.start_date).format("YYYY-MM-DD"))
    const end_date_str = computed(() => moment(props.end_date).format("YYYY-MM-DD"))
    const date_time_str = computed(() => {
        return moment(start_date_str.value).add(slider_model.value, 'seconds').format("MM-DD HH:mm:ss")
    })

    // ── Watchers ──
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

    // pathが更新されたとき中央寄せする
    watch(() => gps_logs.value, () => centering())

    // ── Initialization ──
    update_time_slider_max_value()

    // ── Internal helpers ──

    // datetimeが更新されたとき、sliderの値を更新し、マーカーの位置を更新する。
    function update_time_slider_max_value(): void {
        // 以前はここで開始日から1日ずつ進めて終了日と文字列一致するまで回していたが、
        // 「行き過ぎたら止まる」ガードが無いため次の3つで無限ループになりタブが固まった。
        //   - 終了日が開始日より前 … 終了日に到達しない
        //   - 日付がパース不能 … format() が "Invalid date" を返し永久に一致しない
        //   - 継続条件の !end_date_str.value … 成立したら永久に回る（意図と真逆）
        // 日数差から直接求める。意味は従来どおり (日数差 + 1) * 86400 - 1。
        const start = moment(start_date_str.value, "YYYY-MM-DD", true)
        const end = moment(end_date_str.value, "YYYY-MM-DD", true)
        if (!start.isValid() || !end.isValid() || end.isBefore(start)) {
            // 期間が取れないときは1日ぶんにしておく
            time_slider_max.value = 86400 - 1
            return
        }
        time_slider_max.value = (end.diff(start, 'days') + 1) * 86400 - 1
    }

    async function update_gps_log_lines(): Promise<void> {
        const req = new GetGPSLogRequest()
        // 以前は replace("-", "/") していたが、String.replace は文字列パターンだと
        // 1個目しか置換しないため "2026/08-03" という混在フォーマットになり、
        // momentがネイティブDateパーサへフォールバックしてブラウザ依存になっていた。
        // 書式を明示して日付文字列から直接組み立てる。
        req.start_date = moment(start_date_str.value, "YYYY-MM-DD").startOf('day').toDate()
        req.end_date = moment(end_date_str.value, "YYYY-MM-DD").endOf('day').toDate()
        const res = await props.gkill_api.get_gps_log(req)
        // エラーチェック
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
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
        const datetime = moment(start_date_str.value, "YYYY-MM-DD").startOf('day').add(slider_model.value, 'seconds').toDate().getTime()

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

    async function centering(): Promise<void> {
        if (gps_logs.value.length === 0) {
            return
        }
        let min_lat = 90
        let max_lat = -90
        let min_lng = 180
        let max_lng = -180
        gps_logs.value.forEach(gps_log => {
            if (max_lat < gps_log.latitude.valueOf()) max_lat = gps_log.latitude.valueOf()
            if (min_lat > gps_log.latitude.valueOf()) min_lat = gps_log.latitude.valueOf()
            if (max_lng < gps_log.longitude.valueOf()) max_lng = gps_log.longitude.valueOf()
            if (min_lng > gps_log.longitude.valueOf()) min_lng = gps_log.longitude.valueOf()
        })

        const bounds = {
            north: max_lat,
            south: min_lat,
            east: max_lng,
            west: min_lng,
        }

        gmap.value?.map?.fitBounds(bounds)
        const msec = 100
        center.value = { lat: (min_lat + max_lat) / 2, lng: (min_lng + max_lng) / 2, timestamp: moment().unix() }
        await new Promise(resolve => setTimeout(resolve, msec))
        gmap.value?.map?.fitBounds(bounds)
    }

    // ── Return ──
    return {
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

        // Methods
        centering,
    }
}
