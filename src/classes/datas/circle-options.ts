import type { LatLng } from "./lat-lng"

export interface CircleOptions {
    visible: boolean
    center: LatLng
    radius: number
    strokeColor: string
    strokeOpacity: number
    strokeWeight: number
}