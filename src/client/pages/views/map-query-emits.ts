'use strict'

export interface MapQueryEmits {
    (e: 'request_clear_map_query'): void
    (e: 'request_update_use_map_query', use_map_query: boolean): void
    (e: 'request_update_area', latitude: number, longitude: number, radius: number): void
    (e: 'inited'): void
}
