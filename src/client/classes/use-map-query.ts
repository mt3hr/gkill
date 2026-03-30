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

    // ── Map click handler ──
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    function handle_map_click(event: any): void {
        is_enable_circle.value = true
        latitude.value = event.latLng.lat()
        longitude.value = event.latLng.lng()
        emits('request_update_area', event.latLng.lat(), event.latLng.lng(), radius.value)
    }

    // ── Exposed getters ──
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

    // ── Return ──
    return {
        // State
        query,
        google_map_api_key,
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
