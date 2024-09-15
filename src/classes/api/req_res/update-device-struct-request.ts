'use strict';

import { DeviceStruct } from '@/classes/datas/config/device-struct';
import { GkillAPIRequest } from '../gkill-api-request';


export class UpdateDeviceStructRequest extends GkillAPIRequest {


    device_struct: Array<DeviceStruct>;

    constructor() {
        super()
        this.device_struct = new Array<DeviceStruct>()
    }


}



