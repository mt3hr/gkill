// ˅
'use strict';

import type { GkillError } from '../api/gkill-error';
import { InfoBase } from './info-base';

// ˄

export class Nlog extends InfoBase {
    // ˅

    // ˄

    shop: string;

    title: string;

    amount: Number;

    attached_histories: Array<Nlog>;

    async load_attached_histories(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async clone(): Promise<Nlog> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    constructor() {
        // ˅
        super()
        this.shop = ""
        this.title = ""
        this.amount = 0
        this.attached_histories = new Array<Nlog>()
        // ˄
    }

    // ˅

    // ˄
}

// ˅

// ˄
