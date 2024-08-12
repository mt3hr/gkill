// ˅
'use strict';

import type { GkillError } from '../api/gkill-error';
import { InfoBase } from './info-base';

// ˄

export class Kmemo extends InfoBase {
    // ˅
    
    // ˄

    content: string;

    attached_histories: Array<Kmemo>;

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

    async clone(): Promise<Kmemo> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    constructor() {
        // ˅
        super()
        this.content = ""
        this.attached_histories = new Array<Kmemo>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
