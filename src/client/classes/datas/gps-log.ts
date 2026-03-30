'use strict'

export class GPSLog {

    related_time: Date

    longitude: number

    latitude: number

    constructor() {
        this.related_time = new Date(0)
        this.latitude = 0
        this.longitude = 0
    }

}


