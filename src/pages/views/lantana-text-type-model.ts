'use strict';

import { LantanaTextType } from "@/classes/lantana/lantana-text-type";

export class LantanaTextTypeModel {
    title: string;
    value: LantanaTextType;
    constructor() {
        this.title = ""
        this.value = LantanaTextType.kmemo
    }
}
