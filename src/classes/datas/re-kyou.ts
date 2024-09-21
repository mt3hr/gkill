'use strict';

import type { GkillError } from '../api/gkill-error';
import { InfoBase } from './info-base';
import { Kyou } from './kyou';


export class ReKyou extends InfoBase {


    target_id: string;

    attached_kyou: Kyou;

    async load_attached_kyou(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async load_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async clear_attached_kyou(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async clone(): Promise<ReKyou> {
        throw new Error('Not implemented');
    }

    constructor() {
        super()
        this.target_id = ""
        this.attached_kyou = new Kyou()
    }


}



