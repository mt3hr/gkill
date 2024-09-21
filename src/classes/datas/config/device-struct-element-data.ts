'use strict';


export class DeviceStructElementData {


    id: string;

    device_name: string;

    check_when_inited: boolean;

    children: Array<DeviceStructElementData> | null

    constructor() {
        this.id = ""
        this.device_name = ""
        this.check_when_inited = false
        this.children = null
    }


}



