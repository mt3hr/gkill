'use strict';

import { Text } from '@/classes/datas/text';
import { GkillAPIResponse } from '../gkill-api-response';


export class GetTextHistoryByTagIDResponse extends GkillAPIResponse {


    text_histories: Array<Text>;

    constructor() {
        super()
        this.text_histories = new Array<Text>()
    }


}



