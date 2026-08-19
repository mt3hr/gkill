import { computed, ref, watch, type Ref } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { MapQueryProps } from '@/pages/views/map-query-props'
import type { MapQueryEmits } from '@/pages/views/map-query-emits'

export function useMapQuery(options: {
    props: MapQueryProps,
    emits: MapQueryEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

    const google_map_api_key: Ref<string> = ref(props.gkill_api.get_google_map_api_key())

    // チェックボックスのUI状態。クエリ上は map_latitude/longitude/radius の null 判定
    // （3値そろって有効）が担うため、ローカルrefに分離してprops片方向同期する。
    // サーバ側のゲート(HasMapFilter)も3値そろって初めて有効なので、緯度だけ非nullのような
    // 半端なクエリでチェックが入ると「入っているのに絞られない」表示になる
    function is_map_filter_enabled(query: FindKyouQuery | null | undefined): boolean {
        if (!query) {
            return false
        }
        return query.map_latitude !== null && query.map_longitude !== null && query.map_radius !== null
    }

    const use_map: Ref<boolean> = ref(is_map_filter_enabled(props.find_kyou_query))
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

    // ── Watchers ──
    watch(() => props.find_kyou_query, () => {
        if (!props.find_kyou_query || JSON.stringify(query.value) === JSON.stringify(props.find_kyou_query)) {
            return
        }
        // props同期はユーザー操作ではないのでemitしない。
        // ここでemitすると、フォーカス切替のたびにサイドバーが実検索を発火してループする
        query.value = props.find_kyou_query.clone()
        use_map.value = is_map_filter_enabled(props.find_kyou_query)
        // null着信（フィルタ未使用）ではローカルの座標・半径を既定値のまま保持し、
        // 非null着信のときだけ上書きする
        if (props.find_kyou_query.map_latitude !== null) {
            latitude.value = props.find_kyou_query.map_latitude
        }
        if (props.find_kyou_query.map_longitude !== null) {
            longitude.value = props.find_kyou_query.map_longitude
        }
        is_enable_circle.value = props.find_kyou_query.is_enable_map_circle_in_sidebar
        if (props.find_kyou_query.map_radius !== null) {
            radius.value = props.find_kyou_query.map_radius
        }
    })

    watch(() => props.application_config, async () => {
        emits('inited')
    })

    watch(() => radius.value, () => {
        // v-sliderのv-modelはユーザー操作でも上のprops同期でも書き込まれる。
        // 同期済みクエリと同値なら同期由来なのでemitしない(値比較で判定。タイミングフラグは使わない)
        if (radius.value === query.value.map_radius) {
            return
        }
        emits('request_update_area', latitude.value, longitude.value, radius.value)
    })

    watch(() => props.application_config, () => {
        google_map_api_key.value = props.application_config.google_map_api_key
    })

    // ── Map click handler ──
    function handle_map_click(event: google.maps.MapMouseEvent): void {
        if (!event.latLng) {
            return
        }
        is_enable_circle.value = true
        latitude.value = event.latLng.lat()
        longitude.value = event.latLng.lng()
        emits('request_update_area', event.latLng.lat(), event.latLng.lng(), radius.value)
    }

    // ── Exposed getters ──
    function get_use_map(): boolean {
        return use_map.value
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

    // ── Return ──
    return {
        // State
        query,
        google_map_api_key,
        use_map,
        latitude,
        longitude,
        radius,
        zoom,
        is_enable_circle,
        center,
        circle,

        // Methods
        handle_map_click,
        get_use_map,
        get_latitude,
        get_longitude,
        get_radius,
        get_is_enable_circle,
    }
}
