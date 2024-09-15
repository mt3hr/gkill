'use strict';

export interface MapQueryEmits {
    (e: 'request_clear_map_query'): void
    (e: 'request_update_use_map_query', use_map_query: boolean): void
    (e: 'request update_area', latitude: Number, longitude: Number, radius: Number): void
}