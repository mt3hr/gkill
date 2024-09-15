'use strict';

import type { GkillError } from "./gkill-error";
import type { GkillMessage } from "./gkill-message";


export class GkillAPIResponse {


    messages: Array<GkillMessage>;

    errors: Array<GkillError>;

    constructor() {
        this.messages = new Array<GkillMessage>();
        this.errors = new Array<GkillError>();

    }


}



